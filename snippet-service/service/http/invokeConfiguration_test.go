package http_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	grpcGwPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	snippetTest "github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func invokeConfiguration(ctx context.Context, t *testing.T, id, token string, req *pb.InvokeConfigurationRequest) (*pb.InvokeConfigurationResponse, int, error) {
	data, err := httpTest.GetContentData(&grpcGwPb.Content{
		ContentType: message.AppOcfCbor.String(),
		Data:        hubTest.EncodeToCbor(t, req),
	}, message.AppJSON.String())
	if err != nil {
		return nil, 0, err
	}

	rb := httpTest.NewRequest(http.MethodPost, snippetTest.HTTPURI(snippetHttp.AliasConfigurations), bytes.NewReader(data)).AuthToken(token)
	rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppJSON.String()).ID(id)
	resp := httpTest.Do(t, rb.Build(ctx, t))
	defer func() {
		_ = resp.Body.Close()
	}()

	var got pb.InvokeConfigurationResponse
	err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &got)
	return &got, resp.StatusCode, err
}

func getAppliedConfigurations(ctx context.Context, t *testing.T, snippetClient pb.SnippetServiceClient, req *pb.GetAppliedConfigurationsRequest) (map[string]*pb.AppliedConfiguration, map[string]*pb.AppliedConfiguration_Resource) {
	getClient, errG := snippetClient.GetAppliedConfigurations(ctx, req)
	require.NoError(t, errG)
	defer func() {
		_ = getClient.CloseSend()
	}()
	appliedConfs := make(map[string]*pb.AppliedConfiguration)
	for {
		conf, errR := getClient.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		appliedConfs[conf.GetId()] = conf
	}
	appliedConfResources := make(map[string]*pb.AppliedConfiguration_Resource)
	for _, appliedConf := range appliedConfs {
		for _, r := range appliedConf.GetResources() {
			id := appliedConf.GetConfigurationId().GetId() + "." + r.GetHref()
			appliedConfResources[id] = r
		}
	}
	return appliedConfs, appliedConfResources
}

// wait for applied configurations to get into DONE or TIMEOUT state
func waitForAppliedConfigurations(ctx context.Context, t *testing.T, snippetClient pb.SnippetServiceClient, req *pb.GetAppliedConfigurationsRequest, statusFilter map[string][]pb.AppliedConfiguration_Resource_Status) map[string]*pb.AppliedConfiguration_Resource {
	var appliedConfResources map[string]*pb.AppliedConfiguration_Resource
	retryCount := 0
	for retryCount < 10 {
		_, aConfsResources := getAppliedConfigurations(ctx, t, snippetClient, req)

		for _, r := range aConfsResources {
			sf, ok := statusFilter[r.GetHref()]
			if !ok {
				continue
			}
			if !slices.Contains(sf, r.GetStatus()) {
				goto next
			}
		}
		appliedConfResources = aConfsResources
		break

	next:
		time.Sleep(time.Millisecond * 200) // 2secs total, enough for PendingCommandsCheckInterval to fire multiple times
		retryCount++
	}
	return appliedConfResources
}

func TestRequestHandlerInvokeConfiguration(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		errC := grpcClient.Close()
		require.NoError(t, errC)
	}()
	resources := hubTest.GetAllBackendResourceLinks()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctxWithToken, t, grpcClient.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	defer func() {
		// restore state
		errU := grpcClient.UpdateResource(ctxWithToken, deviceID, hubTest.TestResourceLightInstanceHref("1"), map[string]interface{}{
			"state": false,
			"power": uint64(0),
		}, nil)
		require.NoError(t, errU)
	}()

	snippetCfg := snippetTest.MakeConfig(t)
	snippetCfg.Clients.ResourceAggregate.PendingCommandsCheckInterval = time.Millisecond * 500
	_, shutdownHttp := snippetTest.New(t, snippetCfg)
	defer shutdownHttp()
	logger := log.NewLogger(snippetCfg.Log)

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	snippetClient := pb.NewSnippetServiceClient(conn)

	notExistingResourceHref := "/not/existing"
	canceledResourceHref := "/canceled"
	// configuration1
	// -> /light/1 -> { state: on }
	// -> /not/existing -> { value: 42 }
	conf1, err := snippetClient.CreateConfiguration(ctxWithToken, &pb.Configuration{
		Name:  "update",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: hubTest.TestResourceLightInstanceHref("1"),
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"state": true,
					}),
				},
			},
			{
				Href: notExistingResourceHref,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"value": 42,
					}),
				},
				TimeToLive: int64(100 * time.Millisecond),
			},
			{
				Href: canceledResourceHref,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"level": "leet",
					}),
				},
				TimeToLive: int64(5 * time.Minute),
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf1.GetId())

	// configuration -> /light/1 -> { power: 42 }
	conf2, err := snippetClient.CreateConfiguration(ctxWithToken, &pb.Configuration{
		Name:  "update light power",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: hubTest.TestResourceLightInstanceHref("1"),
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"power": 42,
					}),
				},
				TimeToLive: int64(500 * time.Millisecond),
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf2.GetId())

	logger.Infof("invoke configuration(%v)", conf1.GetId())
	correlationID1 := uuid.NewString()
	got, code, err := invokeConfiguration(ctxWithToken, t, conf1.GetId(), token, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf1.GetId(),
		DeviceId:        deviceID,
		CorrelationId:   correlationID1,
	})
	require.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	light1Conf1ID := conf1.GetId() + "." + hubTest.TestResourceLightInstanceHref("1")
	notExistingConf1ID := conf1.GetId() + "." + notExistingResourceHref
	cancledConf1ID := conf1.GetId() + "." + canceledResourceHref

	appliedConfResources := waitForAppliedConfigurations(ctxWithToken, t, snippetClient,
		&pb.GetAppliedConfigurationsRequest{
			IdFilter: []string{got.GetAppliedConfigurationId()},
		},
		map[string][]pb.AppliedConfiguration_Resource_Status{
			hubTest.TestResourceLightInstanceHref("1"): {pb.AppliedConfiguration_Resource_DONE},
			notExistingResourceHref:                    {pb.AppliedConfiguration_Resource_TIMEOUT},
			canceledResourceHref:                       {pb.AppliedConfiguration_Resource_PENDING},
		},
	)
	require.NotEmpty(t, appliedConfResources)

	canceledConf1, ok := appliedConfResources[cancledConf1ID]
	require.True(t, ok)
	// the second invocation with force should cancel this resource update
	require.Equal(t, pb.AppliedConfiguration_Resource_PENDING, canceledConf1.GetStatus())
	notExistingConf1, ok := appliedConfResources[notExistingConf1ID]
	require.True(t, ok)
	require.Equal(t, pb.AppliedConfiguration_Resource_TIMEOUT, notExistingConf1.GetStatus())
	require.Equal(t, commands.Status_ERROR, notExistingConf1.GetResourceUpdated().GetStatus())
	lightConf1, ok := appliedConfResources[light1Conf1ID]
	require.True(t, ok)
	require.Equal(t, pb.AppliedConfiguration_Resource_DONE, lightConf1.GetStatus())
	require.Equal(t, commands.Status_OK, lightConf1.GetResourceUpdated().GetStatus())

	// /light/1 -> should be updated by invoked conf1
	var gotLight map[interface{}]interface{}
	err = grpcClient.GetResource(ctxWithToken, deviceID, hubTest.TestResourceLightInstanceHref("1"), &gotLight)
	require.NoError(t, err)

	require.Equal(t, map[interface{}]interface{}{
		"state": true,
		"power": uint64(0),
		"name":  "Light",
	}, gotLight)

	logger.Infof("duplicit invoke configuration(%v)", conf1.GetId())
	// duplicit invocation of the same configuration
	correlationID2 := uuid.NewString()
	_, code, err = invokeConfiguration(ctxWithToken, t, conf1.GetId(), token, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf1.GetId(),
		DeviceId:        deviceID,
		CorrelationId:   correlationID2,
	})
	require.Equal(t, http.StatusInternalServerError, code)
	require.Error(t, err)

	logger.Infof("force invoke configuration(%v)", conf1.GetId())
	got2, code, err := invokeConfiguration(ctxWithToken, t, conf1.GetId(), token, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf1.GetId(),
		DeviceId:        deviceID,
		CorrelationId:   correlationID2,
		Force:           true,
	})
	require.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	require.NotEqual(t, got.GetAppliedConfigurationId(), got2.GetAppliedConfigurationId())

	appliedConfResources = waitForAppliedConfigurations(ctxWithToken, t, snippetClient,
		&pb.GetAppliedConfigurationsRequest{
			IdFilter: []string{got2.GetAppliedConfigurationId()},
		},
		map[string][]pb.AppliedConfiguration_Resource_Status{
			hubTest.TestResourceLightInstanceHref("1"): {pb.AppliedConfiguration_Resource_DONE},
			notExistingResourceHref:                    {pb.AppliedConfiguration_Resource_TIMEOUT},
			canceledResourceHref:                       {pb.AppliedConfiguration_Resource_PENDING},
		},
	)
	require.NotEmpty(t, appliedConfResources)

	notExistingConf1, ok = appliedConfResources[notExistingConf1ID]
	require.True(t, ok)
	require.Equal(t, pb.AppliedConfiguration_Resource_TIMEOUT, notExistingConf1.GetStatus())
	require.Equal(t, commands.Status_ERROR, notExistingConf1.GetResourceUpdated().GetStatus())
	lightConf1, ok = appliedConfResources[light1Conf1ID]
	require.True(t, ok)
	require.Equal(t, pb.AppliedConfiguration_Resource_DONE, lightConf1.GetStatus())
	require.Equal(t, commands.Status_OK, lightConf1.GetResourceUpdated().GetStatus())

	appliedConfs, _ := getAppliedConfigurations(ctxWithToken, t, snippetClient, &pb.GetAppliedConfigurationsRequest{
		DeviceIdFilter: []string{deviceID},
	})
	require.Len(t, appliedConfs, 1)
}

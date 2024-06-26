package http_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	grpcGwPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
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

func invokeConfiguration(ctx context.Context, t *testing.T, id, token string, req *pb.InvokeConfigurationRequest) (*pb.AppliedDeviceConfiguration, int, error) {
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

	var got pb.AppliedDeviceConfiguration
	err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &got)
	return &got, resp.StatusCode, err
}

func getAppliedConfigurations(ctx context.Context, t *testing.T, snippetClient pb.SnippetServiceClient) map[string]*pb.AppliedDeviceConfiguration {
	getClient, errG := snippetClient.GetAppliedConfigurations(ctx, &pb.GetAppliedDeviceConfigurationsRequest{})
	require.NoError(t, errG)
	defer func() {
		_ = getClient.CloseSend()
	}()
	appliedConfs := make(map[string]*pb.AppliedDeviceConfiguration)
	for {
		conf, errR := getClient.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		appliedConfs[conf.GetId()] = conf
	}
	return appliedConfs
}

func TestRequestHandlerInvokeConfiguration(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	shutDown := service.SetUpServices(context.Background(), t, service.SetUpServicesOAuth)
	defer shutDown()

	_, shutdownHttp := snippetTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	snippetClient := pb.NewSnippetServiceClient(conn)

	token := oauthTest.GetDefaultAccessToken(t)
	ctxWithToken := pkgGrpc.CtxWithToken(ctx, token)
	notExistingResourceHref := "/not/existing"
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

	correlationID1 := uuid.NewString()
	got, code, err := invokeConfiguration(ctx, t, conf1.GetId(), token, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf1.GetId(),
		DeviceId:        deviceID,
		CorrelationId:   correlationID1,
	})
	require.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	expected := pb.AppliedDeviceConfiguration{
		DeviceId: deviceID,
		ConfigurationId: &pb.AppliedDeviceConfiguration_RelationTo{
			Id:      conf1.GetId(),
			Version: conf1.GetVersion(),
		},
		ExecutedBy: pb.MakeExecutedByOnDemand(),
		Resources: []*pb.AppliedDeviceConfiguration_Resource{
			{
				Href:          hubTest.TestResourceLightInstanceHref("1"),
				Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
				CorrelationId: correlationID1,
			},
			{
				Href:          notExistingResourceHref,
				Status:        pb.AppliedDeviceConfiguration_Resource_PENDING,
				CorrelationId: correlationID1,
			},
		},
		Owner: oauthService.DeviceUserID,
	}

	expected.Id = got.GetId()
	wantCorrelationIDs := make(map[string]string)
	gotCorrelationIDs := make(map[string]string)
	for _, r := range expected.GetResources() {
		wantCorrelationIDs[r.GetHref()] = r.GetCorrelationId()
		r.CorrelationId = ""
	}
	for _, r := range got.GetResources() {
		gotCorrelationIDs[r.GetHref()] = r.GetCorrelationId()
		r.CorrelationId = ""
	}
	require.Len(t, wantCorrelationIDs, len(gotCorrelationIDs))
	for href, wantCorrelationID := range wantCorrelationIDs {
		gotCorrelationID, ok := gotCorrelationIDs[href]
		require.True(t, ok)
		require.True(t, strings.Contains(gotCorrelationID, wantCorrelationID))
	}
	snippetTest.CmpAppliedDeviceConfiguration(t, &expected, got, true)

	// duplicit invocation of the same configuration
	_, code, err = invokeConfiguration(ctx, t, conf1.GetId(), token, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf1.GetId(),
		DeviceId:        deviceID,
		CorrelationId:   correlationID1,
	})
	require.Equal(t, http.StatusInternalServerError, code)
	require.Error(t, err)

	// TODO: force

	appliedConfs := getAppliedConfigurations(pkgGrpc.CtxWithToken(ctx, token), t, snippetClient)
	require.Len(t, appliedConfs, 1)
}

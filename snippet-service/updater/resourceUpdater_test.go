package updater_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestCleanUpExpiredUpdates(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	snippetCfg := test.MakeConfig(t)
	const interval = time.Second
	snippetCfg.Clients.Storage.CleanUpExpiredUpdates = "*/1 * * * * *"
	snippetCfg.Clients.Storage.ExtendCronParserBySeconds = true
	_, shutdownSnippetService := test.New(t, snippetCfg)
	defer shutdownSnippetService()

	token := oauthTest.GetDefaultAccessToken(t)

	snippetClientConn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = snippetClientConn.Close()
	}()
	snippetClient := pb.NewSnippetServiceClient(snippetClientConn)

	ctx = pkgGrpc.CtxWithToken(ctx, token)

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		err = grpcClient.Close()
		require.NoError(t, err)
	}()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctx, t, grpcClient.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, hubTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	notExistingResourceHref := "/not/existing"
	// configuration
	// -> /not/existing -> { value: 42 }
	conf, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
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
	require.NotEmpty(t, conf.GetId())

	// invoke configuration with long TimeToLive
	resp, err := snippetClient.InvokeConfiguration(ctx, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf.GetId(),
		DeviceId:        deviceID,
	})
	require.NoError(t, err)

	time.Sleep(2 * interval) // 2 times the interval to guarantee that the clean up job has run at least once

	// check that all configurations are either in timeout or done state
	s, cleanUpStore := test.NewStore(t)
	defer cleanUpStore()

	appliedConfs := make(map[string]*pb.AppliedConfiguration)
	err = s.GetAppliedConfigurations(ctx, oauthService.DeviceUserID, &pb.GetAppliedConfigurationsRequest{
		IdFilter: []string{resp.GetAppliedConfigurationId()},
	}, func(appliedConf *store.AppliedConfiguration) error {
		appliedConfs[appliedConf.GetId()] = appliedConf.GetAppliedConfiguration().Clone()
		return nil
	})
	require.NoError(t, err)
	require.Len(t, appliedConfs, 1)
	appliedConf, ok := appliedConfs[resp.GetAppliedConfigurationId()]
	require.True(t, ok)
	for _, r := range appliedConf.GetResources() {
		status := r.GetStatus()
		require.True(t, pb.AppliedConfiguration_Resource_TIMEOUT == status || pb.AppliedConfiguration_Resource_DONE == status)
	}
}

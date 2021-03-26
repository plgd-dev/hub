package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/cloud/resource-aggregate/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func TestPublishUnpublish(t *testing.T) {
	config := test.MakeConfig(t)
	config.APIs.GRPC.Server.Addr = "localhost:9888"
	config.APIs.GRPC.Capabilities.SnapshotThreshold = 1

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	authShutdown := authService.SetUp(t)
	defer authShutdown()

	raShutdown := test.New(t, config)
	defer raShutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetServiceToken(t))

	authConn, err := client.New(testCfg.MakeGrpcClientConfig(config.Clients.AuthServer.Addr), log.Get().Desugar())
	require.NoError(t, err)
	defer authConn.Close()
	authClient := pbAS.NewAuthorizationServiceClient(authConn.GRPC())

	raConn, err := client.New(testCfg.MakeGrpcClientConfig(config.APIs.GRPC.Server.Addr), log.Get().Desugar())
	require.NoError(t, err)
	defer raConn.Close()
	raClient := service.NewResourceAggregateClient(raConn.GRPC())

	deviceId := "dev0"
	href := "/oic/p"
	code := oauthTest.GetDeviceAuthorizationCode(t)
	_, err = authClient.SignUp(ctx, &pbAS.SignUpRequest{
		DeviceId:              deviceId,
		AuthorizationCode:     code,
		AuthorizationProvider: "plgd",
	})
	require.NoError(t, err)

	pubReq := testMakePublishResourceRequest(deviceId, []string{href})
	_, err = raClient.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceId, []string{href})
	_, err = raClient.UnpublishResourceLinks(ctx, unpubReq)
	require.NoError(t, err)
}

/*
func testMakePublishResourceRequest(deviceId, href, userId, accesstoken string) *commands.PublishResourceLinksRequest {
	r := &commands.PublishResourceLinksRequest{
		Resources: []*commands.Resource{testNewResource(href, deviceId)},
		DeviceId:  deviceId,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testMakeUnpublishResourceRequest(deviceId, href, userId, accesstoken string) *commands.UnpublishResourceLinksRequest {
	r := &commands.UnpublishResourceLinksRequest{
		Hrefs:    []string{href},
		DeviceId: deviceId,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testNewResource(href string, deviceId string) *commands.Resource {
	return &commands.Resource{
		Href:          href,
		ResourceTypes: []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:    []string{"oic.if.baseline"},
		DeviceId:      deviceId,
		Anchor:        "ocf://" + deviceId + "/oic/p",
		Policies: &commands.Policies{
			BitFlags: 1,
		},
		Title: "device",
	}
}
*/

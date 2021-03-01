package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/kit/security/certManager"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/refImpl"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func TestPublishUnpublish(t *testing.T) {
	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetServiceToken(t))

	authShutdown := authService.SetUp(t)
	defer authShutdown()

	config.Service.AuthServerAddr = testCfg.AUTH_HOST
	config.Service.JwksURL = testCfg.JWKS_URL
	config.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	config.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	config.Service.UserDevicesManagerTickFrequency = time.Millisecond * 500
	config.Service.UserDevicesManagerExpiration = time.Millisecond * 500

	clientCertManager, err := certManager.NewCertManager(config.Dial)
	require.NoError(t, err)
	dialTLSConfig := clientCertManager.GetClientTLSConfig()

	eventstore, err := mongodb.NewEventStore(config.MongoDB, nil, mongodb.WithTLS(dialTLSConfig))
	require.NoError(t, err)
	defer eventstore.Clear(ctx)

	config.Service.Addr = "localhost:9888"
	config.Service.SnapshotThreshold = 1

	server, err := refImpl.Init(config)
	require.NoError(t, err)
	defer server.Shutdown()
	go func() {
		err := server.Serve()
		require.NoError(t, err)
	}()

	authConn, err := grpc.Dial(config.Service.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	require.NoError(t, err)
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	raConn, err := grpc.Dial(config.Service.Addr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	require.NoError(t, err)
	raClient := service.NewResourceAggregateClient(raConn)

	deviceId := "dev0"
	href := "/oic/p"
	code := oauthTest.GetDeviceAuthorizationCode(t)
	resp, err := authClient.SignUp(ctx, &pbAS.SignUpRequest{
		DeviceId:              deviceId,
		AuthorizationCode:     code,
		AuthorizationProvider: "plgd",
	})
	require.NoError(t, err)

	pubReq := testMakePublishResourceRequest(deviceId, href, resp.UserId, resp.AccessToken)
	_, err = raClient.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceId, href, resp.UserId, resp.AccessToken)
	_, err = raClient.UnpublishResourceLinks(ctx, unpubReq)
	require.NoError(t, err)
}

func testMakePublishResourceRequest(deviceId, href, userId, accesstoken string) *commands.PublishResourceLinksRequest {
	r := &commands.PublishResourceLinksRequest{
		Resources:            []*commands.Resource{testNewResource(href, deviceId)},
		DeviceId:             deviceId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId, accesstoken),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testMakeUnpublishResourceRequest(deviceId, href, userId, accesstoken string) *commands.UnpublishResourceLinksRequest {
	r := &commands.UnpublishResourceLinksRequest{
		Hrefs:                []string{href},
		DeviceId:             deviceId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId, accesstoken),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testNewAuthorizationContext(deviceId, userId, accessToken string) *commands.AuthorizationContext {
	ac := commands.AuthorizationContext{
		DeviceId: deviceId,
	}
	return &ac
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

package service_test

import (
	"context"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gofrs/uuid"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	authService "github.com/plgd-dev/cloud/authorization/test"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/refImpl"
	"github.com/plgd-dev/cloud/resource-aggregate/service"

	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func TestPublishUnpublish(t *testing.T) {

	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	oauthShutdown := oauthTest.SetUp(t)
	defer oauthShutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetServiceToken(t))

	authShutdown := authService.SetUp(t)
	defer authShutdown()

	cfg.Clients.Authorization.Addr = testCfg.AUTH_HOST
	cfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	cfg.Clients.OAuthProvider.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Clients.OAuthProvider.OAuth.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	cfg.Service.Grpc.Capabilities.UserDevicesManagerTickFrequency = time.Millisecond * 500
	cfg.Service.Grpc.Capabilities.UserDevicesManagerExpiration = time.Millisecond * 500

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer eventstore.Clear(ctx)

	cfg.Service.Grpc.Addr = "localhost:9888"
	cfg.Service.Grpc.Capabilities.SnapshotThreshold = 1

	svc, err := refImpl.Init(cfg)
	require.NoError(t, err)
	defer svc.Shutdown()
	go func() {
		err := svc.Serve()
		require.NoError(t, err)
	}()

	asCertManager, err := client.New(cfg.Clients.Authorization.TLSConfig, logger)
	require.NoError(t, err)
	authConn, err := grpc.Dial(cfg.Clients.Authorization.Addr, grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())))
	require.NoError(t, err)
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	raCertManager, err := server.New(cfg.Service.Grpc.TLSConfig, logger)
	require.NoError(t, err)
	raConn, err := grpc.Dial(cfg.Service.Grpc.Addr, grpc.WithTransportCredentials(credentials.NewTLS(raCertManager.GetTLSConfig())))
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

package service_test

import (
	"context"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
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
	authProvider "github.com/plgd-dev/cloud/authorization/provider"
	authService "github.com/plgd-dev/cloud/authorization/test"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func TestPublishUnpublish(t *testing.T) {
	ctx := kitNetGrpc.CtxWithToken(context.Background(), authProvider.UserToken)

	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	authShutdown := authService.SetUp(t)
	defer authShutdown()

	cfg.Clients.AuthServer.AuthServerAddr = testCfg.AUTH_HOST
	cfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	cfg.Clients.OAuthProvider.OAuthConfig.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Clients.OAuthProvider.OAuthConfig.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	cfg.Service.RA.Capabilities.UserDevicesManagerTickFrequency = time.Millisecond * 500
	cfg.Service.RA.Capabilities.UserDevicesManagerExpiration = time.Millisecond * 500

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer eventstore.Clear(ctx)

	cfg.Service.RA.GrpcAddr = "localhost:9888"
	cfg.Service.RA.Capabilities.SnapshotThreshold = 1

	service, err := refImpl.Init(cfg)
	require.NoError(t, err)
	defer service.Shutdown()
	go func() {
		err := service.Serve()
		require.NoError(t, err)
	}()

	asCertManager, err := client.New(cfg.Clients.AuthServer.AuthTLSConfig, logger)
	require.NoError(t, err)
	authConn, err := grpc.Dial(cfg.Clients.AuthServer.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(asCertManager.GetTLSConfig())))
	require.NoError(t, err)
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	raCertManager, err := server.New(cfg.Service.RA.GrpcTLSConfig, logger)
	require.NoError(t, err)
	raConn, err := grpc.Dial(cfg.Service.RA.GrpcAddr, grpc.WithTransportCredentials(credentials.NewTLS(raCertManager.GetTLSConfig())))
	require.NoError(t, err)
	raClient := pb.NewResourceAggregateClient(raConn)

	deviceId := "dev0"
	href := "/oic/p"
	resp, err := authClient.SignUp(ctx, &pbAS.SignUpRequest{
		DeviceId:              deviceId,
		AuthorizationCode:     "authcode",
		AuthorizationProvider: authProvider.NewTestProvider().GetProviderName(),
	})
	require.NoError(t, err)

	pubReq := testMakePublishResourceRequest(deviceId, href, resp.UserId, resp.AccessToken)
	_, err = raClient.PublishResource(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceId, href, resp.UserId, resp.AccessToken)
	_, err = raClient.UnpublishResource(ctx, unpubReq)
	require.NoError(t, err)
}

func testMakePublishResourceRequest(deviceId, href, userId, accesstoken string) *pb.PublishResourceRequest {
	r := &pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceId,
			Href:     href,
		},
		Resource:             testNewResource(href, deviceId, utils.MakeResourceId(deviceId, href)),
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId, accesstoken),
		TimeToLive:           1,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testMakeUnpublishResourceRequest(deviceId, href, userId, accesstoken string) *pb.UnpublishResourceRequest {
	r := &pb.UnpublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceId,
			Href:     href,
		},
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId, accesstoken),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testNewAuthorizationContext(deviceId, userId, accessToken string) *pb.AuthorizationContext {
	ac := pb.AuthorizationContext{
		DeviceId: deviceId,
	}
	return &ac
}

func testNewResource(href string, deviceId string, resourceId string) *pb.Resource {
	return &pb.Resource{
		Id:            resourceId,
		Href:          href,
		ResourceTypes: []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:    []string{"oic.if.baseline"},
		DeviceId:      deviceId,
		Anchor:        "ocf://" + deviceId + "/oic/p",
		Policies: &pb.Policies{
			BitFlags: 1,
		},
		Title: "device",
	}
}

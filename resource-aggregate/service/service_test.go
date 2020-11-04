package service_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/kit/security/certificateManager"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gofrs/uuid"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	authProvider "github.com/plgd-dev/cloud/authorization/provider"
	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/kit/config"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

func TestPublishUnpublish(t *testing.T) {
	ctx := kitNetGrpc.CtxWithToken(context.Background(), authProvider.UserToken)

	var cfg refImpl.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	authShutdown := authService.SetUp(t)
	defer authShutdown()

	cfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	cfg.Service.JwksURL = testCfg.JWKS_URL

	clientCertManager, err := certificateManager.NewCertificateManager(cfg.Dial)
	require.NoError(t, err)
	dialTLSConfig := clientCertManager.GetClientTLSConfig()
	eventstore, err := mongodb.NewEventStore(cfg.MongoDB, nil, mongodb.WithTLS(dialTLSConfig))
	require.NoError(t, err)
	defer eventstore.Clear(ctx)

	cfg.Service.Addr = "localhost:9888"
	cfg.Service.SnapshotThreshold = 1

	server, err := refImpl.Init(cfg)
	require.NoError(t, err)
	defer server.Shutdown()
	go func() {
		err := server.Serve()
		require.NoError(t, err)
	}()

	authConn, err := grpc.Dial(cfg.Service.AuthServerAddr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	require.NoError(t, err)
	authClient := pbAS.NewAuthorizationServiceClient(authConn)

	raConn, err := grpc.Dial(cfg.Service.Addr, grpc.WithTransportCredentials(credentials.NewTLS(dialTLSConfig)))
	require.NoError(t, err)
	raClient := pb.NewResourceAggregateClient(raConn)

	deviceId := "dev0"
	resId := "res0"
	resp, err := authClient.SignUp(ctx, &pbAS.SignUpRequest{
		DeviceId:              deviceId,
		AuthorizationCode:     "authcode",
		AuthorizationProvider: authProvider.NewTestProvider().GetProviderName(),
	})
	require.NoError(t, err)

	pubReq := testMakePublishResourceRequest(deviceId, resId, resp.UserId, resp.AccessToken)
	_, err = raClient.PublishResource(ctx, pubReq)
	require.NoError(t, err)

	unpubReq := testMakeUnpublishResourceRequest(deviceId, resId, resp.UserId, resp.AccessToken)
	_, err = raClient.UnpublishResource(ctx, unpubReq)
	require.NoError(t, err)
}

func testMakePublishResourceRequest(deviceId, resourceId, userId, accesstoken string) *pb.PublishResourceRequest {
	href := "/oic/p"
	r := &pb.PublishResourceRequest{
		ResourceId:           resourceId,
		Resource:             testNewResource(href, deviceId, resourceId),
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId, accesstoken),
		TimeToLive:           1,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return r
}

func testMakeUnpublishResourceRequest(deviceId, resourceId, userId, accesstoken string) *pb.UnpublishResourceRequest {
	r := &pb.UnpublishResourceRequest{
		ResourceId:           resourceId,
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
		InstanceId:    1,
		Anchor:        "ocf://" + deviceId + "/oic/p",
		Policies:      &pb.Policies{1},
		Title:         "device",
	}
}

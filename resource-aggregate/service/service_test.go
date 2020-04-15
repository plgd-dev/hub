package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/go-ocf/kit/security/certManager"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	authProvider "github.com/go-ocf/cloud/authorization/provider"
	authConfig "github.com/go-ocf/cloud/authorization/service"
	authService "github.com/go-ocf/cloud/authorization/test/service"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/cloud/resource-aggregate/refImpl"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"
)

func TestPublishUnpublish(t *testing.T) {
	ctx := kitNetGrpc.CtxWithToken(context.Background(), "b")
	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)
	config.Service.AuthServerAddr = "localhost:7000"
	clientCertManager, err := certManager.NewCertManager(config.Dial)
	require.NoError(t, err)
	dialTLSConfig := clientCertManager.GetClientTLSConfig()

	eventstore, err := mongodb.NewEventStore(config.MongoDB, nil, mongodb.WithTLS(dialTLSConfig))
	require.NoError(t, err)
	defer eventstore.Clear(ctx)

	var authConfig authConfig.Config
	envconfig.Process("", &authConfig)
	authConfig.Addr = config.Service.AuthServerAddr

	port := 9888
	authServerShutdown := authService.NewAuthServer(t, authConfig)
	defer authServerShutdown()
	config.Service.Addr = "localhost:" + strconv.Itoa(port)
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
		UserId:   userId,
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

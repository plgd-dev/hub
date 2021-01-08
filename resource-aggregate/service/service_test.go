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

	var config refImpl.Config
	err := envconfig.Process("", &config)
	require.NoError(t, err)

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

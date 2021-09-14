package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/service"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/log"
	cmClient "github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeStorageConfig() service.StorageConfig {
	return service.StorageConfig{
		MongoDB: mongodb.Config{
			URI:      config.MONGODB_URI,
			Database: config.C2C_CONNECTOR_DB,
			TLS:      config.MakeTLSClientConfig(),
		},
	}
}

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.Log.Debug = true

	cfg.APIs.HTTP.EventsURL = config.C2C_CONNECTOR_EVENTS_URL
	cfg.APIs.HTTP.OAuthCallback = config.C2C_CONNECTOR_OAUTH_CALLBACK
	cfg.APIs.HTTP.PullDevices.Disabled = false
	cfg.APIs.HTTP.PullDevices.Interval = time.Second
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.C2C_CONNECTOR_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()

	cfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.Clients.Storage = MakeStorageConfig()
	cfg.Clients.Subscription.HTTP.ReconnectInterval = time.Second * 10
	cfg.Clients.Subscription.HTTP.ResubscribeInterval = time.Second

	cfg.TaskProcessor.CacheSize = 2048
	cfg.TaskProcessor.Timeout = time.Second * 5
	cfg.TaskProcessor.MaxParallel = 128
	cfg.TaskProcessor.Delay = 0

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg.String())
	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

// func setUp(ctx context.Context, t *testing.T, deviceID string, supportedEvents store.Events) func() {
// 	cloud1 := test.SetUp(ctx, t)
// 	cloud2 := c2cConnectorTest.SetUpCloudWithConnector(t)
// 	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

// 	cloud1Conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
// 		RootCAs: test.GetRootCertificatePool(t),
// 	})))
// 	require.NoError(t, err)
// 	c1 := pb.NewGrpcGatewayClient(cloud1Conn)
// 	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c1, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())

func New(t *testing.T, cfg service.Config) func() {
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

	s, err := service.New(context.Background(), cfg, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	return func() {
		_ = s.Shutdown()
		wg.Wait()
	}
}

func NewMongoStore(t *testing.T) (*mongodb.Store, func()) {
	cfg := MakeConfig(t)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

	certManager, err := cmClient.New(cfg.Clients.Storage.MongoDB.TLS, logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := mongodb.NewStore(ctx, cfg.Clients.Storage.MongoDB, certManager.GetTLSConfig())
	require.NoError(t, err)

	cleanUp := func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)
		certManager.Close()
	}

	return store, cleanUp
}

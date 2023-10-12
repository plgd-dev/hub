package test

import (
	"context"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-directory/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.GRPC.Config = config.MakeGrpcServerConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.APIs.GRPC.OwnerCacheExpiration = time.Minute

	cfg.Clients.IdentityStore.Connection = config.MakeGrpcClientConfig(config.IDENTITY_STORE_HOST)

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16

	cfg.Clients.Eventstore.Connection.Use = config.ACTIVE_DATABASE()
	cfg.Clients.Eventstore.Connection.MongoDB = config.MakeEventsStoreMongoDBConfig()
	cfg.Clients.Eventstore.Connection.CqlDB = config.MakeEventsStoreCqlDBConfig()
	cfg.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 120

	cfg.HubID = config.HubID()
	cfg.ExposedHubConfiguration.CAPool = config.CA_POOL
	cfg.ExposedHubConfiguration.Authority = "https://" + config.OAUTH_SERVER_HOST
	cfg.ExposedHubConfiguration.CoapGateway = config.COAP_GW_HOST
	cfg.ExposedHubConfiguration.OwnerClaim = config.OWNER_CLAIM
	cfg.ExposedHubConfiguration.CertificateAuthority = "https://" + config.CERTIFICATE_AUTHORITY_HTTP_HOST

	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	return func() {
		_ = s.Close()
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

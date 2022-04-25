package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-directory/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.GRPC.Config = config.MakeGrpcServerConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.APIs.GRPC.OwnerCacheExpiration = time.Minute

	cfg.Clients.IdentityStore.Connection = config.MakeGrpcClientConfig(config.IDENTITY_STORE_HOST)

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16

	cfg.Clients.Eventstore.Connection.MongoDB = config.MakeEventsStoreMongoDBConfig()
	cfg.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 60

	cfg.ExposedHubConfiguration.CAPool = config.CA_POOL
	cfg.ExposedHubConfiguration.AuthorizationServer = "https://" + config.OAUTH_SERVER_HOST
	cfg.ExposedHubConfiguration.HubID = config.HubID()
	cfg.ExposedHubConfiguration.CoapGateway = config.GW_HOST
	cfg.ExposedHubConfiguration.OwnerClaim = config.OWNER_CLAIM

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)

	s, err := service.New(ctx, cfg, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	return func() {
		s.Close()
		wg.Wait()
	}
}

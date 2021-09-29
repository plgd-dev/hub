package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-directory/service"
	"github.com/plgd-dev/cloud/test/config"

	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.GRPC.Config = config.MakeGrpcServerConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.APIs.GRPC.OwnerCacheExpiration = time.Minute

	cfg.Clients.IdentityServer.CacheExpiration = time.Millisecond * 50
	cfg.Clients.IdentityServer.PullFrequency = time.Millisecond * 200
	cfg.Clients.IdentityServer.Connection = config.MakeGrpcClientConfig(config.IDENTITY_HOST)

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16

	cfg.Clients.Eventstore.Connection.MongoDB = config.MakeEventsStoreMongoDBConfig()
	cfg.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 60

	cfg.ExposedCloudConfiguration.CAPool = config.CA_POOL
	cfg.ExposedCloudConfiguration.AuthorizationServer = "https://" + config.OAUTH_SERVER_HOST
	cfg.ExposedCloudConfiguration.CloudID = config.CloudID()
	cfg.ExposedCloudConfiguration.CloudAuthorizationProvider = config.DEVICE_PROVIDER
	cfg.ExposedCloudConfiguration.CloudURL = config.GW_HOST
	cfg.ExposedCloudConfiguration.OwnerClaim = config.OWNER_CLAIM
	cfg.ExposedCloudConfiguration.SigningServerAddress = config.CERTIFICATE_AUTHORITY_HOST

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

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

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

	cfg.Clients.AuthServer.CacheExpiration = time.Millisecond * 50
	cfg.Clients.AuthServer.PullFrequency = time.Millisecond * 200
	cfg.Clients.AuthServer.OwnerClaim = config.OWNER_CLAIM
	cfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	cfg.Clients.AuthServer.OAuth = config.MakeOAuthConfig()

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16

	cfg.Clients.Eventstore.Connection.MongoDB = config.MakeEventsStoreMongoDBConfig()
	cfg.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 60

	cfg.ExposedCloudConfiguration.CAPool = config.CA_POOL
	cfg.ExposedCloudConfiguration.AuthorizationServer = "https://" + config.OAUTH_SERVER_HOST
	cfg.ExposedCloudConfiguration.CloudID = "cloudID"
	cfg.ExposedCloudConfiguration.CloudAuthorizationProvider = "plgd"
	cfg.ExposedCloudConfiguration.CloudURL = "CloudUrl"
	cfg.ExposedCloudConfiguration.OwnerClaim = "JwtClaimOwnerId"
	cfg.ExposedCloudConfiguration.SigningServerAddress = "SigningServerAddress"

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

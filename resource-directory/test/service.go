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
	var rdCfg service.Config
	rdCfg.APIs.GRPC = config.MakeGrpcServerConfig(config.RESOURCE_DIRECTORY_HOST)

	rdCfg.Clients.AuthServer.CacheExpiration = time.Second
	rdCfg.Clients.AuthServer.PullFrequency = time.Millisecond * 500
	rdCfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	rdCfg.Clients.AuthServer.OAuth = config.MakeOAuthConfig()

	rdCfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()

	rdCfg.Clients.Eventstore.Connection.MongoDB = config.MakeEventsStoreMongoDBConfig()
	rdCfg.Clients.Eventstore.GoPoolSize = 16
	rdCfg.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 60

	rdCfg.ExposedCloudConfiguration.CAPool = config.CA_POOL
	rdCfg.ExposedCloudConfiguration.TokenURL = "AccessTokenUrl"
	rdCfg.ExposedCloudConfiguration.AuthorizationURL = "AuthCodeUrl"
	rdCfg.ExposedCloudConfiguration.CloudID = "cloudID"
	rdCfg.ExposedCloudConfiguration.CloudAuthorizationProvider = "plgd"
	rdCfg.ExposedCloudConfiguration.CloudURL = "CloudUrl"
	rdCfg.ExposedCloudConfiguration.OwnerClaim = "JwtClaimOwnerId"
	rdCfg.ExposedCloudConfiguration.SigningServerAddress = "SigningServerAddress"

	err := rdCfg.Validate()
	require.NoError(t, err)

	return rdCfg
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
		err := s.Serve()
		require.NoError(t, err)
	}()

	return func() {
		s.Close()
		wg.Wait()
	}
}

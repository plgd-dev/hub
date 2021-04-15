package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var raCfg service.Config

	raCfg.APIs.GRPC = testCfg.MakeGrpcServerConfig(testCfg.RESOURCE_AGGREGATE_HOST)

	raCfg.Clients.AuthServer.CacheExpiration = time.Second
	raCfg.Clients.AuthServer.PullFrequency = time.Millisecond * 500
	raCfg.Clients.AuthServer.Connection = testCfg.MakeGrpcClientConfig(testCfg.AUTH_HOST)
	raCfg.Clients.AuthServer.OAuth = testCfg.MakeOAuthConfig()

	raCfg.Clients.Eventbus.NATS = testCfg.MakePublisherConfig()

	raCfg.Clients.Eventstore.Connection.MongoDB = testCfg.MakeEventsStoreMongoDBConfig()
	raCfg.Clients.Eventstore.ConcurrencyExceptionMaxRetry = 8
	raCfg.Clients.Eventstore.SnapshotThreshold = 16

	err := raCfg.Validate()
	require.NoError(t, err)

	return raCfg
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
		s.Shutdown()
		wg.Wait()
	}
}

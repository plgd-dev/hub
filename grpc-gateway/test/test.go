package test

import (
	"context"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.GRPC.Config = config.MakeGrpcServerConfig(config.GRPC_GW_HOST)
	cfg.APIs.GRPC.OwnerCacheExpiration = time.Minute
	cfg.APIs.GRPC.SubscriptionBufferSize = 1000
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false

	cfg.Clients.IdentityStore.Connection = config.MakeGrpcClientConfig(config.IDENTITY_STORE_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
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
	fileWatcher, err := fsnotify.NewWatcher()
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
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}
}

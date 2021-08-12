package test

import (
	"context"
	"sync"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/service"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.GRPC_HOST)
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false

	cfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)

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

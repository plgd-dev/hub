package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-gateway/service"
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeStorageConfig() service.StorageConfig {
	return service.StorageConfig{
		MongoDB: mongodb.Config{
			URI:      config.MONGODB_URI,
			Database: config.C2C_GW_DB,
			TLS:      config.MakeTLSClientConfig(),
		},
	}
}

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.Log.Debug = true

	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.C2C_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.Clients.Storage = MakeStorageConfig()
	cfg.Clients.Subscription.HTTP.ReconnectInterval = time.Second * 10
	cfg.Clients.Subscription.HTTP.EmitEventTimeout = time.Second * 5
	cfg.Clients.Subscription.HTTP.TLS = config.MakeTLSClientConfig()

	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.TaskQueue.MaxIdleTime = time.Minute * 10

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg.String())
	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

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

func NewTestListener(t *testing.T) (net.Listener, func()) {
	loggerCfg := log.Config{Debug: true}
	logger, err := log.NewLogger(loggerCfg)
	require.NoError(t, err)

	listenCfg := config.MakeListenerConfig("localhost:")
	listenCfg.TLS.ClientCertificateRequired = false

	certManager, err := server.New(listenCfg.TLS, logger)
	require.NoError(t, err)

	listener, err := tls.Listen("tcp", listenCfg.Addr, certManager.GetTLSConfig())
	require.NoError(t, err)

	return listener, func() {
		certManager.Close()
	}
}

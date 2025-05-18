package test

import (
	"context"
	"crypto/tls"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
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

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.C2C_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Authorization = config.MakeValidatorConfig()
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_GW_HOST)
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.Storage = MakeStorageConfig()
	cfg.Clients.Subscription.HTTP.ReconnectInterval = time.Second * 10
	cfg.Clients.Subscription.HTTP.EmitEventTimeout = time.Second * 5
	cfg.Clients.Subscription.HTTP.TLS = config.MakeTLSClientConfig()

	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.TaskQueue.MaxIdleTime = time.Minute * 10

	cfg.Clients.OpenTelemetryCollector = http.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, cfg service.Config) func() {
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(context.Background(), cfg, fileWatcher, logger)
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

func C2CURI(uri string) string {
	return testHttp.HTTPS_SCHEME + config.C2C_GW_HOST + uri
}

func GetUniqueSubscriptionID(subIDS ...string) string {
	id := uuid.NewString()
	for slices.Contains(subIDS, id) {
		id = uuid.NewString()
	}
	return id
}

func NewTestListener(t *testing.T) (net.Listener, func()) {
	loggerCfg := log.MakeDefaultConfig()
	logger := log.NewLogger(loggerCfg)

	listenCfg := config.MakeListenerConfig("localhost:")
	listenCfg.TLS.ClientCertificateRequired = false

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	certManager, err := server.New(listenCfg.TLS, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	listener, err := tls.Listen("tcp", listenCfg.Addr, certManager.GetTLSConfig())
	require.NoError(t, err)

	return listener, func() {
		certManager.Close()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

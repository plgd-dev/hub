package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
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

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.C2C_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_HOST)
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
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
	logger := log.NewLogger(cfg.Log)

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

func C2CURI(uri string) string {
	return testHttp.HTTPS_SCHEME + config.C2C_GW_HOST + uri
}

func GetUniqueSubscriptionID(subIDS ...string) string {
	id := uuid.NewString()
	for {
		if !pkgStrings.Contains(subIDS, id) {
			break
		}
		id = uuid.NewString()
	}
	return id
}

func NewTestListener(t *testing.T) (net.Listener, func()) {
	loggerCfg := log.MakeDefaultConfig()
	logger := log.NewLogger(loggerCfg)

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

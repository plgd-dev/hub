package test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/service"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	"github.com/plgd-dev/cloud/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

const (
	OAUTH_MANAGER_CLIENT_ID = oauthService.ClientTest
	OAUTH_MANAGER_AUDIENCE  = "localhost"
)

func MakeAuthorizationConfig() oauth2.Config {
	return oauth2.Config{
		Authority: "https://" + config.OAUTH_SERVER_HOST,
		Config: oauth.Config{
			ClientID:         OAUTH_MANAGER_CLIENT_ID,
			Audience:         OAUTH_MANAGER_AUDIENCE,
			RedirectURL:      config.C2C_CONNECTOR_OAUTH_CALLBACK,
			ClientSecretFile: config.CA_POOL,
		},
		HTTP: config.MakeHttpClientConfig(),
	}
}

func MakeStorageConfig() service.StorageConfig {
	return service.StorageConfig{
		MongoDB: mongodb.Config{
			URI:      config.MONGODB_URI,
			Database: config.C2C_CONNECTOR_DB,
			TLS:      config.MakeTLSClientConfig(),
		},
	}
}

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.Log.Debug = true

	cfg.APIs.HTTP.EventsURL = config.C2C_CONNECTOR_EVENTS_URL
	cfg.APIs.HTTP.PullDevices.Disabled = false
	cfg.APIs.HTTP.PullDevices.Interval = time.Second
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.C2C_CONNECTOR_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Authorization = MakeAuthorizationConfig()

	cfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_HOST)
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.Storage = MakeStorageConfig()
	cfg.Clients.Subscription.HTTP.ReconnectInterval = time.Second * 10
	cfg.Clients.Subscription.HTTP.ResubscribeInterval = time.Second

	cfg.TaskProcessor.CacheSize = 2048
	cfg.TaskProcessor.Timeout = time.Second * 5
	cfg.TaskProcessor.MaxParallel = 128
	cfg.TaskProcessor.Delay = 0

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	cfg := MakeConfig(t)
	fmt.Printf("cfg\n%v\n", cfg.String())
	return New(t, cfg)
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

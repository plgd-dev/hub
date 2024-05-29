package test

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/http-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	"github.com/stretchr/testify/require"
)

func MakeWebConfigurationConfig() service.WebConfiguration {
	return service.WebConfiguration{
		Authority:                 testHttp.HTTPS_SCHEME + config.OAUTH_SERVER_HOST,
		HTTPGatewayAddress:        testHttp.HTTPS_SCHEME + config.HTTP_GW_HOST,
		DeviceProvisioningService: testHttp.HTTPS_SCHEME + config.HTTP_GW_HOST,
		SnippetService:            testHttp.HTTPS_SCHEME + config.SNIPPET_SERVICE_HTTP_HOST,
		WebOAuthClient: service.BasicOAuthClient{
			ClientID: config.OAUTH_MANAGER_CLIENT_ID,
			Audience: config.OAUTH_MANAGER_AUDIENCE,
			Scopes:   []string{"openid", "offline_access"},
		},
		DeviceOAuthClient: service.DeviceOAuthClient{
			BasicOAuthClient: service.BasicOAuthClient{
				ClientID: config.OAUTH_MANAGER_CLIENT_ID,
				Audience: config.OAUTH_MANAGER_AUDIENCE,
				Scopes:   []string{"profile", "openid", "offline_access"},
			},
			ProviderName: config.DEVICE_PROVIDER,
		},
	}
}

func MakeConfig(t require.TestingT, enableUI bool) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.HTTP_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.WebSocket.StreamBodyLimit = 256 * 1024
	cfg.APIs.HTTP.WebSocket.PingFrequency = 10 * time.Second
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()

	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_GW_HOST)
	cfg.Clients.OpenTelemetryCollector = pkgHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}

	if enableUI {
		cfg.UI.Enabled = true
		cfg.UI.Directory = os.Getenv("TEST_HTTP_GW_WWW_ROOT")
		cfg.UI.WebConfiguration = MakeWebConfigurationConfig()
	}

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t, false))
}

func New(t require.TestingT, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
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
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

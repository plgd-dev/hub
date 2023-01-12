package service

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config
	cfg.Log = log.MakeDefaultConfig()
	cfg.Log.DumpBody = true
	cfg.Log.Level = zapcore.DebugLevel
	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.APIs.COAP.Addr = config.COAP_GW_HOST
	cfg.APIs.COAP.RequireBatchObserveEnabled = false
	cfg.APIs.COAP.ExternalAddress = config.COAP_GW_HOST
	cfg.APIs.COAP.Protocols = []coapService.Protocol{coapService.TCP}
	if config.COAP_GATEWAY_UDP_ENABLED {
		cfg.APIs.COAP.Protocols = append(cfg.APIs.COAP.Protocols, coapService.UDP)
	}
	cfg.APIs.COAP.MaxMessageSize = 256 * 1024
	cfg.APIs.COAP.OwnerCacheExpiration = time.Minute
	cfg.APIs.COAP.SubscriptionBufferSize = 1000
	cfg.APIs.COAP.MessagePoolSize = 1000
	cfg.APIs.COAP.KeepAlive = new(coapService.KeepAlive)
	cfg.APIs.COAP.KeepAlive.Timeout = time.Second * 20
	cfg.APIs.COAP.BlockwiseTransfer.Enabled = config.COAP_GATEWAY_UDP_ENABLED
	cfg.APIs.COAP.BlockwiseTransfer.SZX = "1024"
	cfg.APIs.COAP.TLS.Embedded = config.MakeTLSServerConfig()
	cfg.APIs.COAP.TLS.Enabled = new(bool)
	*cfg.APIs.COAP.TLS.Enabled = true
	cfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = false
	cfg.APIs.COAP.TLS.Embedded.CertFile = os.Getenv("TEST_COAP_GW_CERT_FILE")
	cfg.APIs.COAP.TLS.Embedded.KeyFile = os.Getenv("TEST_COAP_GW_KEY_FILE")
	cfg.APIs.COAP.Authorization = service.AuthorizationConfig{
		OwnerClaim: config.OWNER_CLAIM,
		Providers: []service.ProvidersConfig{
			{
				Name:   config.DEVICE_PROVIDER,
				Config: config.MakeDeviceAuthorization(),
			},
		},
	}
	cfg.Clients.IdentityStore.Connection = config.MakeGrpcClientConfig(config.IDENTITY_STORE_HOST)
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

// New creates test coap-gateway.
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
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

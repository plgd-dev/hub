package test

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config
	cfg.Log = config.MakeLogConfig(t, "TEST_COAP_GATEWAY_LOG_LEVEL", "TEST_COAP_GATEWAY_LOG_DUMP_BODY")
	cfg.ServiceHeartbeat.TimeToLive = time.Minute
	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.APIs.COAP.Addr = config.COAP_GW_HOST
	cfg.APIs.COAP.RequireBatchObserveEnabled = false
	cfg.DeviceTwin.MaxETagsCountInRequest = 3
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
	cfg.APIs.COAP.TLS.Embedded.CertFile = urischeme.URIScheme(os.Getenv("TEST_COAP_GW_CERT_FILE"))
	cfg.APIs.COAP.TLS.Embedded.KeyFile = urischeme.URIScheme(os.Getenv("TEST_COAP_GW_KEY_FILE"))
	cfg.APIs.COAP.TLS.Embedded.CRL.Enabled = true
	httpClientConfig := config.MakeHttpClientConfig()
	cfg.APIs.COAP.TLS.Embedded.CRL.HTTP = &httpClientConfig
	cfg.APIs.COAP.Authorization = service.AuthorizationConfig{
		OwnerClaim: config.OWNER_CLAIM,
		Providers: []service.ProvidersConfig{
			{
				Name:   config.DEVICE_PROVIDER,
				Config: config.MakeDeviceAuthorization(),
			},
		},
		Authority: config.MakeValidatorConfig(),
	}
	cfg.Clients.IdentityStore.Connection = config.MakeGrpcClientConfig(config.IDENTITY_STORE_HOST)
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.Clients.CertificateAuthority.Connection = config.MakeGrpcClientConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func checkForClosedSockets(t require.TestingT, cfg service.Config) {
	sockets := make(test.ListenSockets, 0, len(cfg.APIs.COAP.Protocols))
	for _, protocol := range cfg.APIs.COAP.Protocols {
		sockets = append(sockets, test.ListenSocket{
			Network: string(protocol),
			Address: cfg.APIs.COAP.Addr,
		})
	}
	sockets.CheckForClosedSockets(t)
}

// New creates test coap-gateway.
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

		checkForClosedSockets(t, cfg)
	}
}

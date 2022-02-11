package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.Log.Config = log.MakeDefaultConfig()

	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.APIs.COAP.Addr = config.GW_HOST
	cfg.APIs.COAP.ExternalAddress = config.GW_HOST
	cfg.APIs.COAP.MaxMessageSize = 256 * 1024
	cfg.APIs.COAP.OwnerCacheExpiration = time.Minute
	cfg.APIs.COAP.SubscriptionBufferSize = 1000
	cfg.APIs.COAP.MessagePoolSize = 1000
	cfg.APIs.COAP.KeepAlive.Timeout = time.Second * 20
	cfg.APIs.COAP.BlockwiseTransfer.Enabled = false
	cfg.APIs.COAP.BlockwiseTransfer.SZX = "1024"
	cfg.APIs.COAP.TLS.Embedded = config.MakeTLSServerConfig()
	cfg.APIs.COAP.TLS.Enabled = true
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

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("config %v\n", cfg.String())

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

// New creates test coap-gateway.
func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log.Config)

	s, err := service.New(ctx, cfg, logger)
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
	}
}

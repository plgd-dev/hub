package service

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.Log.DumpCoapMessages = true
	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024
	cfg.APIs.COAP.Addr = config.GW_HOST
	cfg.APIs.COAP.ExternalAddress = config.GW_HOST
	cfg.APIs.COAP.MaxMessageSize = 256 * 1024
	cfg.APIs.COAP.GoroutineSocketHeartbeat = time.Millisecond * 300
	cfg.APIs.COAP.KeepAlive.Timeout = time.Second * 20
	cfg.APIs.COAP.BlockwiseTransfer.Enabled = true
	cfg.APIs.COAP.BlockwiseTransfer.SZX = "1024"
	cfg.APIs.COAP.TLS.Embedded = config.MakeTLSServerConfig()
	cfg.APIs.COAP.TLS.Enabled = true
	cfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = false
	cfg.APIs.COAP.TLS.Embedded.CertFile = os.Getenv("TEST_COAP_GW_CERT_FILE")
	cfg.APIs.COAP.TLS.Embedded.KeyFile = os.Getenv("TEST_COAP_GW_KEY_FILE")
	cfg.APIs.COAP.Authorization.Domain = "https://" + config.OAUTH_SERVER_HOST
	cfg.APIs.COAP.Authorization.ClientID = oauthService.ClientTest
	cfg.APIs.COAP.Authorization.HTTP = config.MakeHttpClientConfig()
	cfg.Clients.AuthServer.OwnerClaim = "sub"
	cfg.Clients.AuthServer.Connection = config.MakeGrpcClientConfig(config.AUTH_HOST)
	cfg.Clients.AuthServer.OAuth = config.MakeOAuthConfig()
	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)
	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

// New creates test coap-gateway.
func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log.Embedded)
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
		_ = s.Close()
		wg.Wait()
	}
}

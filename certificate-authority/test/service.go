package test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.APIs.HTTP.Addr = config.CERTIFICATE_AUTHORITY_HTTP_HOST
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	cfg.Signer.KeyFile = os.Getenv("TEST_ROOT_CA_KEY")
	cfg.Signer.CertFile = os.Getenv("TEST_ROOT_CA_CERT")
	cfg.Signer.ValidFrom = "now-1h"
	cfg.Signer.ExpiresIn = time.Hour * 2
	cfg.Signer.HubID = config.HubID()

	cfg.Log = log.MakeDefaultConfig()
	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
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

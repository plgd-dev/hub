package test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	cfg.Signer.KeyFile = os.Getenv("TEST_ROOT_CA_KEY")
	cfg.Signer.CertFile = os.Getenv("TEST_ROOT_CA_CERT")
	cfg.Signer.ValidFrom = "now-1h"
	cfg.Signer.ExpiresIn = time.Hour * 2
	cfg.Signer.HubID = config.HubID()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log)
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
		s.Close()
		wg.Wait()
	}
}

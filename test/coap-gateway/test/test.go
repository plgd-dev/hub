package test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.Log.Config.Level = zapcore.DebugLevel
	cfg.Log.DumpCoapMessages = true
	cfg.APIs.COAP.Addr = config.GW_HOST
	cfg.APIs.COAP.TLS.Config = config.MakeTLSServerConfig()
	cfg.APIs.COAP.TLS.Config.ClientCertificateRequired = false
	cfg.APIs.COAP.TLS.Config.CertFile = os.Getenv("TEST_COAP_GW_CERT_FILE")
	cfg.APIs.COAP.TLS.Config.KeyFile = os.Getenv("TEST_COAP_GW_KEY_FILE")
	cfg.APIs.COAP.TLS.Enabled = true
	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t *testing.T, makeHandler service.MakeServiceHandler, verifyOnClose service.VerifyServiceHandler) (TearDown func()) {
	return New(t, MakeConfig(t), makeHandler, verifyOnClose)
}

func New(t *testing.T, cfg service.Config, makeHandler service.MakeServiceHandler, verifyOnClose service.VerifyServiceHandler) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log.Config)

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger, makeHandler)
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
		if verifyOnClose == nil {
			return
		}
		for _, c := range s.GetClients() {
			verifyOnClose(c.GetServiceHandler())
		}
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

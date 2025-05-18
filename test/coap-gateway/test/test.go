package test

import (
	"context"
	"os"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config
	cfg.Log.Level = log.DebugLevel
	cfg.Log.DumpCoapMessages = true
	cfg.APIs.COAP.Addr = config.COAP_GW_HOST
	cfg.APIs.COAP.TLS.Config = config.MakeTLSServerConfig()
	cfg.APIs.COAP.TLS.ClientCertificateRequired = false
	cfg.APIs.COAP.TLS.CertFile = urischeme.URIScheme(os.Getenv("TEST_COAP_GW_CERT_FILE"))
	cfg.APIs.COAP.TLS.KeyFile = urischeme.URIScheme(os.Getenv("TEST_COAP_GW_KEY_FILE"))
	cfg.APIs.COAP.TLS.Enabled = true
	cfg.TaskQueue.GoPoolSize = 1600
	cfg.TaskQueue.Size = 2 * 1024 * 1024

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT, getHandler service.GetServiceHandler, onShutdown service.OnShutdown) (tearDown func()) {
	return New(t, MakeConfig(t), getHandler, onShutdown)
}

func New(t require.TestingT, cfg service.Config, getHandler service.GetServiceHandler, onShutdown service.OnShutdown) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log.Config)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger, getHandler)
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
		if onShutdown != nil {
			for _, c := range s.GetClients() {
				onShutdown(c.GetServiceHandler())
			}
		}

		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

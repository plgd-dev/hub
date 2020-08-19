package test

import (
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) refImpl.Config {
	var cfg refImpl.Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	cfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	cfg.Service.ResourceAggregateAddr = testCfg.RESOURCE_AGGREGATE_HOST
	cfg.Service.Addr = testCfg.C2C_CONNECTOR_HOST
	cfg.Service.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	cfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	cfg.Service.OAuthCallback = testCfg.C2C_CONNECTOR_OAUTH_CALLBACK
	cfg.Service.EventsURL = testCfg.C2C_CONNECTOR_EVENTS_URL
	cfg.Service.JwksURL = testCfg.JWKS_URL
	cfg.Listen.File.DisableVerifyClientCertificate = true
	cfg.Service.PullDevicesInterval = time.Second
	cfg.Service.ResubscribeInterval = time.Second
	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

// NewC2CConnector creates test c2c-connector.
func New(t *testing.T, cfg refImpl.Config) func() {
	t.Log("newC2CConnector")
	defer t.Log("newC2CConnector done")
	c, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		c.Serve()
	}()

	return func() {
		c.Shutdown()
		wg.Wait()
	}
}

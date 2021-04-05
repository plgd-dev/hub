package test

import (
	"github.com/plgd-dev/cloud/cloud2cloud-connector/service"
	"github.com/plgd-dev/kit/config"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	cfg.Clients.Authorization.Addr = testCfg.AUTH_HOST
	cfg.Clients.ResourceAggregate.Addr = testCfg.RESOURCE_AGGREGATE_HOST
	cfg.Service.Http.Addr = testCfg.C2C_CONNECTOR_HOST
	cfg.Clients.ResourceDirectory.Addr = testCfg.RESOURCE_DIRECTORY_HOST
	cfg.Clients.OAuthProvider.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Clients.OAuthProvider.OAuth.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	cfg.Service.Http.OAuthCallback = testCfg.C2C_CONNECTOR_OAUTH_CALLBACK
	cfg.Service.Http.EventsURL = testCfg.C2C_CONNECTOR_EVENTS_URL
	cfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	cfg.Service.Http.TLSConfig.ClientCertificateRequired = false
	cfg.Service.Capabilities.PullDevicesInterval = time.Second
	cfg.Service.Capabilities.ResubscribeInterval = time.Second
	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

// NewC2CConnector creates test c2c-connector.
func New(t *testing.T, cfg service.Config) func() {
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

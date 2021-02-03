package test

import (
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/service"
	"github.com/plgd-dev/kit/config"
	"sync"
	"testing"

	"github.com/plgd-dev/cloud/cloud2cloud-gateway/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	err := config.Load(&cfg)
	require.NoError(t, err)

	cfg.Service.Http.Addr = testCfg.C2C_GW_HOST
	cfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	cfg.Clients.ResourceDirectory.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	cfg.Service.Http.FQDN = "cloud2cloud-gateway-" + t.Name()
	cfg.Service.Http.TLSConfig.ClientCertificateRequired = false
	cfg.Clients.OAuthProvider.OAuthConfig.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	cfg.Clients.OAuthProvider.OAuthConfig.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {

	s, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve()
	}()

	return func() {
		s.Shutdown()
		wg.Wait()
	}
}

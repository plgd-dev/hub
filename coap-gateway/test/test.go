package service

import (
	"sync"
	"testing"

	"github.com/go-ocf/cloud/coap-gateway/refImpl"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func SetUp(t *testing.T) (TearDown func()) {
	var gwCfg refImpl.Config
	err := envconfig.Process("", &gwCfg)
	require.NoError(t, err)
	gwCfg.Service.Addr = testCfg.GW_HOST
	gwCfg.Service.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	gwCfg.Service.FQDN = "coap-gateway-" + t.Name()
	gwCfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	gwCfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	return NewCoapGateway(t, gwCfg)
}

// NewCoapGateway creates test coap-gateway.
func NewCoapGateway(t *testing.T, cfg refImpl.Config) func() {
	t.Log("newCoapGateway")
	defer t.Log("newCoapGateway done")
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

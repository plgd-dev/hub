package service

import (
	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/kit/config"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T, withoutTLS ...bool) service.Config {
	var gwCfg service.Config
	err := config.Load(&gwCfg)
	require.NoError(t, err)

	if len(withoutTLS) > 0 {
		gwCfg.Service.CoapGW.ServerTLSConfig.ClientCertificateRequired = false
		gwCfg.Service.CoapGW.Addr = testCfg.GW_UNSECURE_HOST
	} else {
		gwCfg.Service.CoapGW.ServerTLSConfig.ClientCertificateRequired = true
		gwCfg.Service.CoapGW.Addr = testCfg.GW_HOST
	}
	gwCfg.Clients.Authorization.AuthServerAddr = testCfg.AUTH_HOST
	gwCfg.Clients.ResourceAggregate.ResourceAggregateAddr = testCfg.RESOURCE_AGGREGATE_HOST
	gwCfg.Clients.ResourceDirectory.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	gwCfg.Service.CoapGW.ExternalAddress = "coap-gateway-" + t.Name()
	gwCfg.Clients.OAuthProvider.OAuthConfig.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	gwCfg.Clients.OAuthProvider.OAuthConfig.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	gwCfg.Service.CoapGW.HeartBeat = time.Millisecond * 300

	gwCfg.Service.CoapGW.ServerTLSConfig.CertFile = os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_FILE_CERT_NAME")
	gwCfg.Service.CoapGW.ServerTLSConfig.KeyFile = os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_FILE_KEY_NAME")
	gwCfg.Service.CoapGW.ServerTLSConfig.ClientCertificateRequired = false
	gwCfg.Log.Debug = true
	return gwCfg
}

func SetUp(t *testing.T, withoutTLS ...bool) (TearDown func()) {
	return New(t, MakeConfig(t, withoutTLS...))
}

// New creates test coap-gateway.
func New(t *testing.T, cfg service.Config) func() {

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

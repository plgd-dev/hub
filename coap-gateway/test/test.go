package service

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/coap-gateway/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T, withoutTLS ...bool) refImpl.Config {
	var gwCfg refImpl.Config
	err := envconfig.Process("", &gwCfg)
	require.NoError(t, err)
	if len(withoutTLS) > 0 {
		gwCfg.ListenWithoutTLS = true
		gwCfg.Service.Addr = testCfg.GW_UNSECURE_HOST
	} else {
		gwCfg.ListenWithoutTLS = false
		gwCfg.Service.Addr = testCfg.GW_HOST
	}
	gwCfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	gwCfg.Service.ResourceAggregateAddr = testCfg.RESOURCE_AGGREGATE_HOST

	gwCfg.Service.ResourceDirectoryAddr = testCfg.RESOURCE_DIRECTORY_HOST
	gwCfg.Service.FQDN = "coap-gateway-" + t.Name()
	gwCfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	gwCfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	gwCfg.Service.HeartBeat = time.Millisecond * 300

	gwCfg.Listen.File.TLSCertFileName = os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_FILE_CERT_NAME")
	gwCfg.Listen.File.TLSKeyFileName = os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_FILE_KEY_NAME")
	gwCfg.Listen.File.DisableVerifyClientCertificate = true
	return gwCfg
}

func SetUp(t *testing.T, withoutTLS ...bool) (TearDown func()) {
	return New(t, MakeConfig(t, withoutTLS...))
}

// New creates test coap-gateway.
func New(t *testing.T, cfg refImpl.Config) func() {

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

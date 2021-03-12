package test

import (
	"sync"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/resource-directory/refImpl"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) refImpl.Config {
	var rdCfg refImpl.Config
	err := envconfig.Process("", &rdCfg)
	require.NoError(t, err)
	rdCfg.Addr = testCfg.RESOURCE_DIRECTORY_HOST
	rdCfg.Service.AuthServerAddr = testCfg.AUTH_HOST
	rdCfg.Service.FQDN = "resource-directory-" + t.Name()
	rdCfg.Service.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	rdCfg.Service.OAuth.Endpoint.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Service.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE
	rdCfg.UserDevicesManagerTickFrequency = time.Millisecond * 500
	rdCfg.UserDevicesManagerExpiration = time.Millisecond * 500
	rdCfg.JwksURL = testCfg.JWKS_URL
	rdCfg.Log.Debug = true
	return rdCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg refImpl.Config) func() {

	s, err := refImpl.Init(cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Serve()
		require.NoError(t, err)
	}()

	return func() {
		s.Close()
		wg.Wait()
	}
}

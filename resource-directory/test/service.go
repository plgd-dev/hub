package test

import (
	"github.com/plgd-dev/kit/config"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/resource-directory/refImpl"
	"github.com/plgd-dev/cloud/resource-directory/service"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var rdCfg service.Config
	err := config.Load(&rdCfg)
	require.NoError(t, err)
	rdCfg.Service.Grpc.Addr = testCfg.RESOURCE_DIRECTORY_HOST
	rdCfg.Service.Grpc.FQDN = "resource-directory-" + t.Name()
	rdCfg.Clients.Authorization.Addr = testCfg.AUTH_HOST
	rdCfg.Clients.ResourceAggregate.Addr = testCfg.RESOURCE_AGGREGATE_HOST
	rdCfg.Clients.OAuthProvider.OAuth.ClientID = testCfg.OAUTH_MANAGER_CLIENT_ID
	rdCfg.Clients.OAuthProvider.OAuth.TokenURL = testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Clients.OAuthProvider.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE
	rdCfg.Service.Grpc.Capabilities.UserDevicesManagerTickFrequency = time.Millisecond * 500
	rdCfg.Service.Grpc.Capabilities.UserDevicesManagerExpiration = time.Millisecond * 500
	rdCfg.Clients.OAuthProvider.JwksURL = testCfg.JWKS_URL
	rdCfg.Log.Debug = true
	return rdCfg
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
		err := s.Serve()
		require.NoError(t, err)
	}()

	return func() {
		s.Shutdown()
		wg.Wait()
	}
}

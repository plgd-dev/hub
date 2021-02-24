package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/authorization/service"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/kit/security/certManager"
)

func newService(config service.Config, tlsConfig *tls.Config) (*service.Server, error) {
	oauth := provider.NewAuth0Provider(config.Device, tlsConfig)
	persistence, err := mongodb.NewStore(context.Background(), config.MongoDB, mongodb.WithTLS(tlsConfig))
	if err != nil {
		return nil, err
	}

	s, err := service.New(config, persistence, oauth, oauth)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}

	return s, nil
}

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config
	err := envconfig.Process("", &authCfg)
	require.NoError(t, err)
	authCfg.Addr = testCfg.AUTH_HOST
	authCfg.HTTPAddr = testCfg.AUTH_HTTP_HOST
	authCfg.Device.Provider = "auth0"
	authCfg.Device.OAuth2.ClientID = oauthService.ClientTest
	authCfg.Device.OAuth2.Endpoint.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.Device.OAuth2.Endpoint.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.SDK.ClientID = oauthService.ClientTest
	authCfg.SDK.Endpoint.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	return authCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, config service.Config) func() {
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()

	auth, err := newService(config, tlsConfig)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = auth.Serve()
		require.NoError(t, err)
	}()

	return func() {
		auth.Shutdown()
		dialCertManager.Close()
		wg.Wait()
	}
}

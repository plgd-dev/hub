package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/authorization/service"
	testCfg "github.com/plgd-dev/cloud/test/config"

	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/kit/security/certManager/server"
)

func newService(config service.Config, tlsConfig *tls.Config) (*service.Server, error) {
	logger, err := log.NewLogger(config.Log)

	oauth := provider.NewPlgdProvider(config.Clients.Device, tlsConfig)
	persistence, err := mongodb.NewStore(context.Background(), config.Database.MongoDB, mongodb.WithTLS(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	s, err := service.New(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}

	return s, nil
}

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config
	err := config.Load(&authCfg)
	require.NoError(t, err)
	authCfg.Service.Grpc.Addr = testCfg.AUTH_HOST
	authCfg.Service.Http.Addr = testCfg.AUTH_HTTP_HOST
	authCfg.Clients.Device.Provider = "plgd"
	authCfg.Clients.Device.OAuth.ClientID = oauthService.ClientTest
	authCfg.Clients.Device.OAuth.Endpoint.AuthURL = "https://" + testCfg.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.Clients.Device.OAuth.Endpoint.TokenURL = "https://" + testCfg.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.SDK.OAuth.ClientID = oauthService.ClientTest
	authCfg.Clients.SDK.OAuth.Endpoint.TokenURL = "https://" + testCfg.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.SDK.OAuth.Audience = testCfg.OAUTH_MANAGER_AUDIENCE
	return authCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, config service.Config) func() {
	logger, err := log.NewLogger(config.Log)
	assert.NoError(t, err)
	if err != nil {
		fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	dialCertManager, err := server.New(config.Service.Grpc.TLSConfig, logger)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetTLSConfig()

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
		wg.Wait()
	}
}

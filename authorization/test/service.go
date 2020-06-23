package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"

	"github.com/go-ocf/cloud/authorization/persistence/mongodb"
	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/authorization/service"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/kit/security/certManager"
)

func newService(config service.Config, tlsConfig *tls.Config) (*service.Server, error) {
	oauth := provider.NewTestProvider()
	persistence, err := mongodb.NewStore(context.Background(), config.MongoDB, mongodb.WithTLS(tlsConfig))
	if err != nil {
		return nil, err
	}

	s, err := service.New(config, persistence, oauth, oauth)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %v", err)
	}

	return s, nil
}

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config
	err := envconfig.Process("", &authCfg)
	require.NoError(t, err)
	authCfg.Addr = testCfg.AUTH_HOST
	authCfg.HTTPAddr = testCfg.AUTH_HTTP_HOST
	authCfg.Device.Provider = "test"
	return authCfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return NewAuthServer(t, MakeConfig(t))
}

func NewAuthServer(t *testing.T, config service.Config) func() {
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

package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-ocf/ocf-cloud/authorization/persistence/mongodb"
	"github.com/go-ocf/ocf-cloud/authorization/provider"
	"github.com/go-ocf/ocf-cloud/authorization/service"
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

func NewAuthServer(t *testing.T, config service.Config) func() {
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()

	auth, err := newService(config, &tlsConfig)
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

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID      = "testUserID"
	testDeviceID    = "testDeviceID"
	testAccessToken = "testAccessToken"
)

func makeConfig(t *testing.T) Config {
	var authCfg Config

	authCfg.APIs.GRPC.Addr = config.AUTH_HOST
	authCfg.APIs.GRPC.TLS.CAPool = config.CA_POOL
	authCfg.APIs.GRPC.TLS.CertFile = config.CERT_FILE
	authCfg.APIs.GRPC.TLS.KeyFile = config.KEY_FILE
	authCfg.APIs.GRPC.Authorization = config.MakeAuthorizationConfig()
	authCfg.APIs.HTTP.TLS.ClientCertificateRequired = true

	authCfg.APIs.HTTP.Addr = config.AUTH_HTTP_HOST
	authCfg.APIs.HTTP.TLS.CAPool = config.CA_POOL
	authCfg.APIs.HTTP.TLS.CertFile = config.CERT_FILE
	authCfg.APIs.HTTP.TLS.KeyFile = config.KEY_FILE
	authCfg.APIs.HTTP.TLS.ClientCertificateRequired = false

	authCfg.OAuthClients.Device.Provider = "plgd"
	authCfg.OAuthClients.Device.ClientID = oauthService.ClientTest
	authCfg.OAuthClients.Device.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.OAuthClients.Device.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.OAuthClients.Device.HTTP = config.MakeHttpClientConfig()

	authCfg.OAuthClients.SDK.ClientID = oauthService.ClientTest
	authCfg.OAuthClients.SDK.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.OAuthClients.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	authCfg.OAuthClients.SDK.HTTP = config.MakeHttpClientConfig()

	authCfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	authCfg.Clients.Storage.MongoDB.Database = "ownersDevices"
	authCfg.Clients.Storage.MongoDB.TLS.CAPool = config.CA_POOL
	authCfg.Clients.Storage.MongoDB.TLS.CertFile = config.CERT_FILE
	authCfg.Clients.Storage.MongoDB.TLS.KeyFile = config.KEY_FILE

	err := authCfg.Validate()
	require.NoError(t, err)

	return authCfg
}

func newTestService(t *testing.T) (*Server, func()) {
	return newTestServiceWithProviders(t, nil, nil)
}

func newTestServiceWithProviders(t *testing.T, deviceProvider, sdkProvider Provider) (*Server, func()) {
	cfg := makeConfig(t)
	cfg.OAuthClients.Device.ClientID = "test client id"
	cfg.OAuthClients.Device.ClientSecret = "test client secret"

	if deviceProvider == nil {
		deviceProvider = NewTestProvider()
	}
	if sdkProvider == nil {
		deviceProvider = NewTestProvider()
	}

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

	s, err := NewServer(context.Background(), cfg, logger, deviceProvider, sdkProvider)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.Serve()
		defer wg.Done()
	}()
	return s, func() {
		s.Shutdown()
		wg.Wait()
	}
}

func (s *Server) cleanUp() {
	p := s.service.persistence
	p.Clear(context.Background())
	p.Close(context.Background())
}

func newTestDevice() *persistence.AuthorizedDevice {
	return &persistence.AuthorizedDevice{
		DeviceID:     testDeviceID,
		Owner:        testUserID,
		AccessToken:  testAccessToken,
		RefreshToken: "testRefreshToken",
		Expiry:       time.Now().Add(time.Hour),
	}
}

func retrieveDevice(t *testing.T, p Persistence, deviceID, userID string) (d *persistence.AuthorizedDevice, ok bool) {
	tx := p.NewTransaction(context.Background())
	defer tx.Close()
	d, ok, err := tx.Retrieve(deviceID, userID)
	assert.Nil(t, err)
	return
}

func persistDevice(t *testing.T, p Persistence, d *persistence.AuthorizedDevice) {
	tx := p.NewTransaction(context.Background())
	defer tx.Close()
	err := tx.Persist(d)
	assert.Nil(t, err)
}

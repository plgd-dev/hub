package service

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-ocf/cloud/authorization/persistence"
	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/cloud/authorization/persistence/mongodb"
	oauthProvider "github.com/go-ocf/cloud/authorization/provider"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID      = "testUserID"
	testDeviceID    = "testDeviceID"
	testAccessToken = "testAccessToken"
)

func newTestService(t *testing.T) (*Server, func()) {
	return newTestServiceWithProviders(t, nil, nil)
}

func newTestServiceWithProviders(t *testing.T, deviceProvider, sdkProvider Provider) (*Server, func()) {
	os.Setenv("CLIENTID", "test client id")
	os.Setenv("CLIENTSECRET", "test client secret")
	os.Setenv("OAUTHENDPOINTDOMAIN", "")
	os.Setenv("CALLBACKURL", "")

	var cfg Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)

	t.Log(cfg.String())
	dialCertManager, err := certManager.NewCertManager(cfg.Dial)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()

	if deviceProvider == nil {
		deviceProvider = oauthProvider.NewTestProvider()
	}
	if sdkProvider == nil {
		deviceProvider = oauthProvider.NewTestProvider()
	}
	persistence, err := mongodb.NewStore(context.Background(), cfg.MongoDB, mongodb.WithTLS(&tlsConfig))
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "gotesttmp")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	s, err := New(cfg, persistence, deviceProvider, sdkProvider)
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
		UserID:       testUserID,
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

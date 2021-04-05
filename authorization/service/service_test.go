package service

import (
	"context"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/authorization/persistence/mongodb"
	"github.com/plgd-dev/kit/security/certManager/client"
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
	var cfg Config
	err := config.Load(&cfg)
	require.NoError(t, err)
	t.Log(cfg.String())

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	mongoCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	tlsConfig := mongoCertManager.GetTLSConfig()

	if deviceProvider == nil {
		deviceProvider = NewTestProvider()
	}
	if sdkProvider == nil {
		deviceProvider = NewTestProvider()
	}
	persistence, err := mongodb.NewStore(context.Background(), cfg.Database.MongoDB, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "gotesttmp")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	s, err := New(cfg)
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

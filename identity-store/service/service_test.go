package service

import (
	"context"
	"sync"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

const (
	testUserID   = "testUserID"
	testDeviceID = "testDeviceID"
	testUser2    = "testUser2"
)

func makeConfig(t *testing.T) Config {
	var cfg Config

	cfg.APIs.GRPC.Addr = config.IDENTITY_STORE_HOST
	cfg.APIs.GRPC.TLS.CAPool = config.CA_POOL
	cfg.APIs.GRPC.TLS.CertFile = config.CERT_FILE
	cfg.APIs.GRPC.TLS.KeyFile = config.KEY_FILE
	cfg.APIs.GRPC.Authorization.OwnerClaim = config.OWNER_CLAIM
	cfg.APIs.GRPC.Authorization.Config = config.MakeAuthorizationConfig()

	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.Database = config.IDENTITY_STORE_DB
	cfg.Clients.Storage.MongoDB.TLS.CAPool = config.CA_POOL
	cfg.Clients.Storage.MongoDB.TLS.CertFile = config.CERT_FILE
	cfg.Clients.Storage.MongoDB.TLS.KeyFile = config.KEY_FILE

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func newTestService(t *testing.T) (*Server, func()) {
	cfg := makeConfig(t)

	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)

	naClient, publisher, err := test.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger)
	require.NoError(t, err)

	s, err := NewServer(context.Background(), cfg, fileWatcher, logger, trace.NewNoopTracerProvider(), publisher)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = s.Serve()
		defer wg.Done()
	}()
	return s, func() {
		_ = s.Close()
		publisher.Close()
		naClient.Close()
		wg.Wait()
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}
}

func (s *Server) cleanUp() error {
	p := s.service.persistence
	var errors *multierror.Error
	if err := p.Clear(context.Background()); err != nil {
		errors = multierror.Append(errors, err)
	}
	if err := p.Close(context.Background()); err != nil {
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

func newTestDevice() *persistence.AuthorizedDevice {
	return newTestDeviceWithIDAndOwner(testDeviceID, testUserID)
}

func newTestDeviceWithIDAndOwner(deviceID, owner string) *persistence.AuthorizedDevice {
	return &persistence.AuthorizedDevice{
		DeviceID: deviceID,
		Owner:    owner,
	}
}

func persistDevice(t *testing.T, p Persistence, d *persistence.AuthorizedDevice) {
	tx := p.NewTransaction(context.Background())
	defer tx.Close()
	err := tx.Persist(d)
	assert.Nil(t, err)
}

package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/cqldb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	testUserID = "testUserID"
	testUser2  = "testUser2"
)

var testDeviceID = test.GenerateDeviceIDbyIdx(123456)

func MakeConfig(t require.TestingT) Config {
	var cfg Config

	cfg.APIs.GRPC.Addr = config.IDENTITY_STORE_HOST
	cfg.APIs.GRPC.TLS.CAPool = config.CA_POOL
	cfg.APIs.GRPC.TLS.CertFile = config.CERT_FILE
	cfg.APIs.GRPC.TLS.KeyFile = config.KEY_FILE
	cfg.APIs.GRPC.Authorization.OwnerClaim = config.OWNER_CLAIM
	cfg.APIs.GRPC.Authorization.Config = config.MakeValidatorConfig()

	cfg.HubID = config.HubID()

	cfg.Clients.Storage.Use = config.ACTIVE_DATABASE()
	cfg.Clients.Storage.MongoDB = &mongodb.Config{}
	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	cfg.Clients.Storage.MongoDB.Database = config.IDENTITY_STORE_DB
	cfg.Clients.Storage.CqlDB = &cqldb.Config{}
	cfg.Clients.Storage.CqlDB.Embedded = config.MakeCqlDBConfig()

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig(t)

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func newTestService(t *testing.T) (*Server, func()) {
	cfg := MakeConfig(t)

	fmt.Printf("cfg: %v\n", cfg.String())

	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	s, err := NewServer(context.Background(), cfg, fileWatcher, logger, noop.NewTracerProvider(), publisher)
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
	require.NoError(t, err)
}

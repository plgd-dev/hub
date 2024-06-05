package cqldb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/plgd-dev/hub/v2/integration-service/store"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/otel/trace"
)

// Document
const (
	// cqldb has all keys in lowercase
	idKey         = "id"
	ownerKey      = "owner"
	deviceIDKey   = "deviceid"
	commonNameKey = "commonname"
	dataKey       = "data"
)

type Index struct {
	Name            string
	PartitionKey    string
	SecondaryColumn string
}

// partition key: idKey
// clustering key: deviceIDKey
var primaryKey = []string{idKey, ownerKey, commonNameKey}

// Store implements an Store for cqldb.
type Store struct {
	client *cqldb.Client
	config *Config
	logger log.Logger
}

func New(ctx context.Context, config *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(config.Embedded.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}
	cqldbClient, err := cqldb.New(ctx, config.Embedded, certManager.GetTLSConfig(), logger, tracerProvider)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	store, err := newEventStoreWithClient(ctx, cqldbClient, config, logger)
	if err != nil {
		cqldbClient.Close()
		certManager.Close()
		return nil, err
	}
	store.AddCloseFunc(certManager.Close)
	return store, nil
}

func createEventsTable(ctx context.Context, client *cqldb.Client, table string) error {
	q := "create table if not exists " + client.Keyspace() + "." + table + " (" +
		idKey + " " + cqldb.UUIDType + "," +
		ownerKey + " " + cqldb.StringType + "," +
		deviceIDKey + " " + cqldb.UUIDType + "," +
		commonNameKey + " " + cqldb.StringType + "," +
		dataKey + " " + cqldb.BytesType + "," +
		"primary key (" + strings.Join(primaryKey, ",") + ")" +
		")"
	err := client.Session().Query(q).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table(%v): %w", table, err)
	}
	return nil
}

// NewEventStoreWithClient creates a new Store with a session.
func newEventStoreWithClient(ctx context.Context, client *cqldb.Client, config *Config, logger log.Logger) (*Store, error) {
	if client == nil {
		return nil, errors.New("invalid client")
	}

	if config.Table == "" {
		config.Table = "integrations"
	}

	err := createEventsTable(ctx, client, config.Table)
	if err != nil {
		return nil, err
	}

	return &Store{
		client: client,
		logger: logger,
		config: config,
	}, nil
}

// Clear clears the event storage.
func (s *Store) Clear(ctx context.Context) error {
	err := s.client.DropKeyspace(ctx)
	if err != nil {
		return fmt.Errorf("cannot clear: %w", err)
	}
	return nil
}

func (s *Store) Table() string {
	return s.client.Keyspace() + "." + s.config.Table
}

// Clear documents in collections, but don't drop the database or the collections
func (s *Store) ClearTable(ctx context.Context) error {
	return s.client.Session().Query("truncate " + s.Table() + ";").WithContext(ctx).Exec()
}

// Close closes the database session.
func (s *Store) Close(_ context.Context) error {
	s.client.Close()
	return nil
}

func (s *Store) AddCloseFunc(f func()) {
	s.client.AddCloseFunc(f)
}

func (s *Store) DeleteExpiredRecords(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

func (s *Store) CreateRecord(ctx context.Context, r *store.ConfigurationRecord) error {
	return nil
}

package cqldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/otel/trace"
)

// Document
const (
	// cqldbdb has all keys in lowercase
	ownerKey    = "ownerkey"
	deviceIDKey = "deviceid"
)

// partition key: deviceIDKey
var primaryKey = []string{deviceIDKey}

var indexes = []cqldb.Index{
	{
		Name:            "ownerIndex",
		SecondaryColumn: ownerKey,
	},
}

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
		deviceIDKey + " " + cqldb.UUIDType + "," +
		ownerKey + " " + cqldb.StringType + "," +
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
		config.Table = "deviceOwners"
	}

	err := createEventsTable(ctx, client, config.Table)
	if err != nil {
		return nil, err
	}
	err = client.CreateIndexes(ctx, config.Table, indexes)
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

// Truncate records in table, but don't drop the keybase or the table.
func (s *Store) TruncateTable(ctx context.Context) error {
	return s.client.Session().Query("TRUNCATE TABLE " + s.Table()).WithContext(ctx).Exec()
}

// Close closes the database session.
func (s *Store) Close(_ context.Context) error {
	s.client.Close()
	return nil
}

func (s *Store) AddCloseFunc(f func()) {
	s.client.AddCloseFunc(f)
}

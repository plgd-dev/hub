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

type Store struct {
	*cqldb.Store
}

func New(ctx context.Context, config *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(config.Embedded.TLS, fileWatcher, logger, tracerProvider)
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
		config.Table = "signedCertificateRecords"
	}

	err := createEventsTable(ctx, client, config.Table)
	if err != nil {
		return nil, err
	}

	return &Store{
		Store: cqldb.NewStore(config.Table, client, logger),
	}, nil
}

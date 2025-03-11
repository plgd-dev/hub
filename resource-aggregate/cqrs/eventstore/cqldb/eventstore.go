package cqldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"go.opentelemetry.io/otel/trace"
)

// Document
const (
	// cqldbdb has all keys in lowercase
	idKey            = "id"
	versionKey       = "version"
	snapshotKey      = "snapshot"
	serviceIDKey     = "serviceid"
	deviceIDKey      = "deviceid"
	etagKey          = "etag"
	timestampKey     = "timestamp"
	etagTimestampKey = "etagtimestamp"
	eventTypeKey     = "eventtype"
)

// partition key: deviceIDKey
// clustering key: idKey
var primaryKey = []string{deviceIDKey, idKey}

var indexes = []cqldb.Index{
	{
		Name:            "serviceIndex",
		SecondaryColumn: serviceIDKey,
	},
}

// EventStore implements an EventStore for cqldb.
type EventStore struct {
	*cqldb.Store
	marshalerFunc   MarshalerFunc
	unmarshalerFunc UnmarshalerFunc
}

func New(ctx context.Context, config *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...Option) (*EventStore, error) {
	config.marshalerFunc = json.Marshal
	config.unmarshalerFunc = json.Unmarshal
	for _, o := range opts {
		o.apply(config)
	}
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
		versionKey + " " + cqldb.Int64Type + "," +
		deviceIDKey + " " + cqldb.UUIDType + "," +
		serviceIDKey + " " + cqldb.UUIDType + "," +
		etagKey + " " + cqldb.BytesType + "," +
		snapshotKey + " " + cqldb.BytesType + "," +
		timestampKey + " " + cqldb.Int64Type + "," +
		etagTimestampKey + " " + cqldb.Int64Type + "," +
		eventTypeKey + " " + cqldb.StringType + "," +
		"primary key (" + strings.Join(primaryKey, ",") + ")" +
		")"
	err := client.Session().Query(q).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to create table(%v): %w", table, err)
	}
	return nil
}

// NewEventStoreWithClient creates a new EventStore with a session.
func newEventStoreWithClient(ctx context.Context, client *cqldb.Client, config *Config, logger log.Logger) (*EventStore, error) {
	if client == nil {
		return nil, errors.New("invalid client")
	}

	if config.marshalerFunc == nil {
		return nil, errors.New("no event marshaler")
	}
	if config.unmarshalerFunc == nil {
		return nil, errors.New("no event unmarshaler")
	}

	if config.Table == "" {
		config.Table = "events"
	}

	err := createEventsTable(ctx, client, config.Table)
	if err != nil {
		return nil, err
	}

	err = client.CreateIndexes(ctx, config.Table, indexes)
	if err != nil {
		return nil, err
	}

	return &EventStore{
		Store:           cqldb.NewStore(config.Table, client, logger),
		marshalerFunc:   config.marshalerFunc,
		unmarshalerFunc: config.unmarshalerFunc,
	}, nil
}

func encodeToBlob(data []byte) string {
	var b strings.Builder
	cqldb.EncodeToBlob(data, &b)
	return b.String()
}

func getLatestEventsSnapshot(events []eventstore.Event, marshaler MarshalerFunc) (eventstore.Event, []byte, error) {
	if len(events) == 0 {
		return nil, nil, errors.New("empty events")
	}
	lastEvent := events[len(events)-1]
	if !lastEvent.IsSnapshot() {
		return nil, nil, errors.New("the last event must be a snapshot")
	}
	// Marshal event data if there is any.
	snapshot, err := marshaler(lastEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot marshal snapshot event: %w", err)
	}
	return lastEvent, snapshot, nil
}

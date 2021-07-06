package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

const eventCName = "events"

// Event
const versionKey = "version"
const dataKey = "data"
const eventTypeKey = "eventtype"
const isSnapshotKey = "issnapshot"
const timestampKey = "timestamp"

// Document
const aggregateIDKey = "aggregateid"
const idKey = "_id"
const firstVersionKey = "firstversion"
const latestVersionKey = "latestversion"
const latestSnapshotVersionKey = "latestsnapshotversion"
const latestTimestampKey = "latesttimestamp"
const eventsKey = "events"
const groupIDKey = "groupid"
const isActiveKey = "isactive"

var aggregateIDLastVersionQueryIndex = bson.D{
	{Key: aggregateIDKey, Value: 1},
	{Key: latestVersionKey, Value: 1},
}

var aggregateIDFirstVersionQueryIndex = bson.D{
	{Key: aggregateIDKey, Value: 1},
	{Key: firstVersionKey, Value: 1},
}

var groupIDQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: isActiveKey, Value: 1},
}

var groupIDaggregateIDQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: aggregateIDKey, Value: 1},
	{Key: isActiveKey, Value: 1},
}

type signOperator string

const (
	signOperator_gte signOperator = "$gte"
	signOperator_lt  signOperator = "$lt"
)

type LogDebugfFunc = func(fmt string, args ...interface{})

//MarshalerFunc marshal struct to bytes.
type MarshalerFunc = func(v interface{}) ([]byte, error)

//UnmarshalerFunc unmarshal bytes to pointer of struct.
type UnmarshalerFunc = func(b []byte, v interface{}) error

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	client          *mongo.Client
	LogDebugfFunc   LogDebugfFunc
	dbPrefix        string
	colPrefix       string
	batchSize       int
	dataMarshaler   MarshalerFunc
	dataUnmarshaler UnmarshalerFunc
	ensuredIndexes  *cache.Cache
	closeFunc       []func()
}

func (s *EventStore) AddCloseFunc(f func()) {
	s.closeFunc = append(s.closeFunc, f)
}

func New(ctx context.Context, config Config, logger *zap.Logger, opts ...Option) (*EventStore, error) {
	config.marshalerFunc = json.Marshal
	config.unmarshalerFunc = json.Unmarshal
	for _, o := range opts {
		o.apply(&config)
	}
	certManager, err := client.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI).SetMaxPoolSize(config.MaxPoolSize).SetMaxConnIdleTime(config.MaxConnIdleTime).SetTLSConfig(certManager.GetTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}

	store, err := newEventStoreWithClient(ctx, client, config.Database, "events", config.BatchSize, config.marshalerFunc, config.unmarshalerFunc, nil)
	if err != nil {
		return nil, err
	}
	store.AddCloseFunc(certManager.Close)
	return store, nil
}

// NewEventStore creates a new EventStore.
//lint:ignore U1000 Lets keep this for now
func newEventStore(ctx context.Context, host, dbPrefix string, colPrefix string, batchSize int, eventMarshaler MarshalerFunc, eventUnmarshaler UnmarshalerFunc, LogDebugfFunc LogDebugfFunc, opts ...*options.ClientOptions) (*EventStore, error) {
	newOpts := []*options.ClientOptions{options.Client().ApplyURI("mongodb://" + host)}
	newOpts = append(newOpts, opts...)
	client, err := mongo.Connect(ctx, newOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("could not dial database: %w", err)
	}

	s, err := newEventStoreWithClient(ctx, client, dbPrefix, colPrefix, batchSize, eventMarshaler, eventUnmarshaler, LogDebugfFunc)
	if err != nil {
		client.Disconnect(ctx)
		return nil, err
	}
	return s, err
}

// NewEventStoreWithClient creates a new EventStore with a session.
func newEventStoreWithClient(ctx context.Context, client *mongo.Client, dbPrefix string, colPrefix string, batchSize int, eventMarshaler MarshalerFunc, eventUnmarshaler UnmarshalerFunc, LogDebugfFunc LogDebugfFunc) (*EventStore, error) {
	if client == nil {
		return nil, errors.New("invalid client")
	}

	if eventMarshaler == nil {
		return nil, errors.New("no event marshaler")
	}
	if eventUnmarshaler == nil {
		return nil, errors.New("no event unmarshaler")
	}

	if dbPrefix == "" {
		dbPrefix = "default"
	}

	if dbPrefix == "" {
		colPrefix = "events"
	}

	if batchSize < 1 {
		batchSize = 128
	}

	if LogDebugfFunc == nil {
		LogDebugfFunc = func(fmt string, args ...interface{}) {}
	}

	s := &EventStore{
		client:          client,
		dbPrefix:        dbPrefix,
		colPrefix:       colPrefix,
		dataMarshaler:   eventMarshaler,
		dataUnmarshaler: eventUnmarshaler,
		batchSize:       batchSize,
		LogDebugfFunc:   LogDebugfFunc,
		ensuredIndexes:  cache.New(time.Hour, time.Hour),
	}

	colAv := s.client.Database(s.DBName()).Collection(maintenanceCName)
	err := s.ensureIndex(ctx, colAv)
	if err != nil {
		return nil, fmt.Errorf("cannot save maintenance query: %w", err)
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())
	err = s.ensureIndex(ctx,
		col,
		aggregateIDLastVersionQueryIndex,
		aggregateIDFirstVersionQueryIndex,
		groupIDQueryIndex,
		groupIDaggregateIDQueryIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot save events: %w", err)
	}

	return s, nil
}

func (s *EventStore) ensureIndex(ctx context.Context, col *mongo.Collection, indexes ...bson.D) error {
	_, ok := s.ensuredIndexes.Get(col.Name())
	if ok {
		return nil
	}
	for _, keys := range indexes {
		opts := options.Index()
		index := mongo.IndexModel{
			Keys:    keys,
			Options: opts,
		}

		_, err := col.Indexes().CreateOne(ctx, index)
		if err != nil {
			if strings.HasPrefix(err.Error(), "(IndexKeySpecsConflict)") {
				//index already exist, just skip error and continue
				continue
			}
			return fmt.Errorf("cannot ensure indexes for eventstore: %w", err)
		}
	}
	s.ensuredIndexes.SetDefault(col.Name(), true)
	return nil
}

func getEventCollectionName() string {
	return "devices_" + eventCName
}

func getDocID(event eventstore.Event) string {
	return fmt.Sprintf("%v.%v", event.AggregateID(), event.Version())
}

func getLatestSnapshotVersion(events []eventstore.Event) (uint64, error) {
	err := fmt.Errorf("not found")
	var latestSnapshotVersion uint64
	for _, e := range events {
		if e.IsSnapshot() {
			latestSnapshotVersion = e.Version()
			err = nil
		}
	}
	if err != nil && len(events) > 0 {
		if events[0].Version() == 0 {
			latestSnapshotVersion = 0
			err = nil
		}
	}
	return latestSnapshotVersion, err
}

func makeDBDoc(events []eventstore.Event, marshaler MarshalerFunc) (bson.M, error) {
	e, err := makeDBEvents(events, marshaler)
	if err != nil {
		return nil, fmt.Errorf("cannot insert first events('%v'): %w", events, err)
	}
	latestSnapshotVersion, err := getLatestSnapshotVersion(events)
	if err != nil {
		return nil, fmt.Errorf("cannot get latestSnapshotVersion from events('%v'): %w", events, err)
	}
	return bson.M{
		idKey:                    getDocID(events[0]),
		groupIDKey:               events[0].GroupID(),
		aggregateIDKey:           events[0].AggregateID(),
		latestVersionKey:         events[len(events)-1].Version(),
		firstVersionKey:          events[0].Version(),
		latestSnapshotVersionKey: latestSnapshotVersion,
		latestTimestampKey:       events[len(events)-1].Timestamp().UnixNano(),
		isActiveKey:              true,
		eventsKey:                e,
	}, nil
}

// DBName returns db name
func (s *EventStore) DBName() string {
	ns := "db"
	return s.dbPrefix + "_" + ns
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	err := s.client.Database(s.DBName()).Drop(ctx)
	if err != nil {
		return fmt.Errorf("cannot clear: %w", err)
	}

	return nil
}

// Close closes the database session.
func (s *EventStore) Close(ctx context.Context) error {
	s.ensuredIndexes.Flush()
	err := s.client.Disconnect(ctx)
	for _, f := range s.closeFunc {
		f()
	}
	return err
}

// newDBEvent returns a new dbEvent for an eventstore.
func makeDBEvents(events []eventstore.Event, marshaler MarshalerFunc) ([]bson.M, error) {
	dbEvents := make([]bson.M, 0, len(events))
	for idx, event := range events {
		// Marshal event data if there is any.
		raw, err := marshaler(event)
		if err != nil {
			return nil, fmt.Errorf("cannot create db event from event[%v]: %w", idx, err)
		}
		dbEvents = append(dbEvents, bson.M{
			versionKey:    event.Version(),
			dataKey:       raw,
			eventTypeKey:  event.EventType(),
			isSnapshotKey: event.IsSnapshot(),
			timestampKey:  pkgTime.UnixNano(event.Timestamp()),
		})
	}
	return dbEvents, nil
}

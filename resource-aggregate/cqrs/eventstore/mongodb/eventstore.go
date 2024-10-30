package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

const eventCName = "events"

// Event
const versionKey = "version"

const (
	dataKey       = "data"
	eventTypeKey  = "eventtype"
	isSnapshotKey = "issnapshot"
	timestampKey  = "timestamp"
)

// Document
const aggregateIDKey = "aggregateid"

const (
	idKey                     = "_id"
	firstVersionKey           = "firstversion"
	latestVersionKey          = "latestversion"
	latestSnapshotVersionKey  = "latestsnapshotversion"
	latestTimestampKey        = "latesttimestamp"
	eventsKey                 = "events"
	groupIDKey                = "groupid"
	isActiveKey               = "isactive"
	latestETagKey             = "latestetag"
	etagKey                   = "etag"
	typesKey                  = "types"
	latestETagKeyTimestampKey = latestETagKey + "." + timestampKey
	serviceIDKey              = "serviceid"
)

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

var serviceIDQueryIndex = bson.D{
	{Key: serviceIDKey, Value: 1},
	{Key: isActiveKey, Value: 1},
}

var groupIDaggregateIDQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: aggregateIDKey, Value: 1},
	{Key: isActiveKey, Value: 1},
}

var groupIDLatestTimestampQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: latestTimestampKey, Value: 1},
}

var aggregateIDLatestTimestampQueryIndex = bson.D{
	{Key: aggregateIDKey, Value: 1},
	{Key: latestTimestampKey, Value: 1},
}

var groupIDETagLatestTimestampQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: latestETagKeyTimestampKey, Value: -1},
}

var groupIDTypesQueryIndex = bson.D{
	{Key: groupIDKey, Value: 1},
	{Key: typesKey, Value: 1},
	{Key: isActiveKey, Value: 1},
}

type signOperator string

const (
	signOperator_gte signOperator = "$gte"
	signOperator_lt  signOperator = "$lt"
)

type LogDebugfFunc = func(fmt string, args ...interface{})

// MarshalerFunc marshal struct to bytes.
type MarshalerFunc = func(v interface{}) ([]byte, error)

// UnmarshalerFunc unmarshal bytes to pointer of struct.
type UnmarshalerFunc = func(b []byte, v interface{}) error

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	store           *pkgMongo.Store
	LogDebugfFunc   LogDebugfFunc
	dbPrefix        string
	colPrefix       string
	dataMarshaler   MarshalerFunc
	dataUnmarshaler UnmarshalerFunc
	ensuredIndexes  *cache.Cache[string, bool]
}

func (s *EventStore) AddCloseFunc(f func()) {
	s.store.AddCloseFunc(f)
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
	mgoStore, err := pkgMongo.NewStore(ctx, &config.Embedded, certManager.GetTLSConfig(), tracerProvider)
	if err != nil {
		return nil, err
	}
	store, err := newEventStoreWithClient(ctx, mgoStore, config.Embedded.Database, "events", config.marshalerFunc, config.unmarshalerFunc, nil)
	if err != nil {
		return nil, err
	}
	store.AddCloseFunc(certManager.Close)
	return store, nil
}

// NewEventStoreWithClient creates a new EventStore with a session.
func newEventStoreWithClient(ctx context.Context, store *pkgMongo.Store, dbPrefix string, colPrefix string, eventMarshaler MarshalerFunc, eventUnmarshaler UnmarshalerFunc, logDebugfFunc LogDebugfFunc) (*EventStore, error) {
	if store == nil {
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

	if colPrefix == "" {
		colPrefix = "events"
	}

	if logDebugfFunc == nil {
		logDebugfFunc = func(string, ...interface{}) {
			// no-op if not set
		}
	}
	ensuredIndexes := cache.NewCache[string, bool]()
	add := periodic.New(ctx.Done(), time.Hour/2)
	add(func(now time.Time) bool {
		ensuredIndexes.CheckExpirations(now)
		return true
	})

	s := &EventStore{
		store:           store,
		dbPrefix:        dbPrefix,
		colPrefix:       colPrefix,
		dataMarshaler:   eventMarshaler,
		dataUnmarshaler: eventUnmarshaler,
		LogDebugfFunc:   logDebugfFunc,
		ensuredIndexes:  ensuredIndexes,
	}

	colAv := s.client().Database(s.DBName()).Collection(maintenanceCName)
	err := s.ensureIndex(ctx, colAv)
	if err != nil {
		return nil, fmt.Errorf("cannot save maintenance query: %w", err)
	}

	col := s.client().Database(s.DBName()).Collection(getEventCollectionName())
	err = s.ensureIndex(ctx,
		col,
		aggregateIDLastVersionQueryIndex,
		aggregateIDFirstVersionQueryIndex,
		groupIDQueryIndex,
		groupIDaggregateIDQueryIndex,
		groupIDLatestTimestampQueryIndex,
		aggregateIDLatestTimestampQueryIndex,
		groupIDETagLatestTimestampQueryIndex,
		serviceIDQueryIndex,
		groupIDTypesQueryIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot save events: %w", err)
	}

	return s, nil
}

func (s *EventStore) client() *mongo.Client {
	return s.store.Client()
}

func (s *EventStore) ensureIndex(ctx context.Context, col *mongo.Collection, indexes ...bson.D) error {
	v := s.ensuredIndexes.Load(col.Name())
	if v != nil {
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
				// index already exist, just skip error and continue
				continue
			}
			return fmt.Errorf("cannot ensure indexes for eventstore: %w", err)
		}
	}
	s.ensuredIndexes.LoadOrStore(col.Name(), cache.NewElement(true, time.Now().Add(time.Hour), nil))
	return nil
}

func getEventCollectionName() string {
	return "devices_" + eventCName
}

func getDocID(event eventstore.Event) string {
	return fmt.Sprintf("%v.%v", event.AggregateID(), event.Version())
}

func getLatestSnapshotVersion(events []eventstore.Event) (uint64, error) {
	err := errors.New("not found")
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

func makeDBETag(etag *eventstore.ETagData) bson.M {
	if etag == nil {
		return nil
	}
	return bson.M{
		etagKey:      etag.ETag,
		timestampKey: etag.Timestamp,
	}
}

func tryToSetServiceID(doc bson.M, events []eventstore.Event) bson.M {
	for _, e := range events {
		serviceID, ok := e.ServiceID()
		if ok {
			doc[serviceIDKey] = serviceID
		}
	}
	return doc
}

func makeDBDoc(events []eventstore.Event, marshaler MarshalerFunc) (bson.M, error) {
	etag, types, e, err := makeDBEventsAndGetETag(events, marshaler)
	if err != nil {
		return nil, fmt.Errorf("cannot insert first events('%v'): %w", events, err)
	}
	latestSnapshotVersion, err := getLatestSnapshotVersion(events)
	if err != nil {
		return nil, fmt.Errorf("cannot get latestSnapshotVersion from events('%v'): %w", events, err)
	}
	d := bson.M{
		idKey:                    getDocID(events[0]),
		groupIDKey:               events[0].GroupID(),
		aggregateIDKey:           events[0].AggregateID(),
		latestVersionKey:         events[len(events)-1].Version(),
		firstVersionKey:          events[0].Version(),
		latestSnapshotVersionKey: latestSnapshotVersion,
		latestTimestampKey:       events[len(events)-1].Timestamp().UnixNano(),
		isActiveKey:              true,
		eventsKey:                e,
		latestETagKey:            makeDBETag(etag),
	}
	if etag != nil {
		d[etagKey] = etag.ETag
	}
	if len(types) > 0 {
		d[typesKey] = types
	}
	return tryToSetServiceID(d, events), nil
}

// DBName returns db name
func (s *EventStore) DBName() string {
	ns := "db"
	return s.dbPrefix + "_" + ns
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	err := s.client().Database(s.DBName()).Drop(ctx)
	if err != nil {
		return fmt.Errorf("cannot clear: %w", err)
	}

	return nil
}

// Clear documents in collections, but don't drop the database or the collections
func (s *EventStore) ClearCollections(ctx context.Context) error {
	cols, err := s.client().Database(s.DBName()).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to obtain collection names: %w", err)
	}
	var errors *multierror.Error
	for _, col := range cols {
		if _, err2 := s.client().Database(s.DBName()).Collection(col).DeleteMany(ctx, bson.D{}); err2 != nil {
			errors = multierror.Append(errors, fmt.Errorf("failed to clear collection %v: %w", col, err2))
		}
	}
	return errors.ErrorOrNil()
}

// Close closes the database session.
func (s *EventStore) Close(ctx context.Context) error {
	_ = s.ensuredIndexes.LoadAndDeleteAll()
	return s.store.Close(ctx)
}

// newDBEvent returns a new dbEvent for an eventstore.
func makeDBEventsAndGetETag(events []eventstore.Event, marshaler MarshalerFunc) (*eventstore.ETagData, []string, []bson.M, error) {
	dbEvents := make([]bson.M, 0, len(events))
	var etag *eventstore.ETagData
	var types []string
	for idx, event := range events {
		// Marshal event data if there is any.
		raw, err := marshaler(event)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot create db event from event[%v]: %w", idx, err)
		}
		dbEvents = append(dbEvents, bson.M{
			versionKey:    event.Version(),
			dataKey:       raw,
			eventTypeKey:  event.EventType(),
			isSnapshotKey: event.IsSnapshot(),
			timestampKey:  pkgTime.UnixNano(event.Timestamp()),
		})
		et := event.ETag()
		if et != nil {
			etag = et
		}
		if len(event.Types()) > 0 {
			types = event.Types()
		}
	}
	return etag, types, dbEvents, nil
}

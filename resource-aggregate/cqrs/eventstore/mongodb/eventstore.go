package mongodb

import (
	"context"
	"errors"
	"fmt"
	"hash/crc64"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
)

const eventCName = "events"

const aggregateIDKey = "aggregateid"
const aggregateIDStrKey = "aggregateidstr"
const idKey = "_id"
const versionKey = "version"
const dataKey = "data"
const eventTypeKey = "eventtype"

var snapshotsQueryIndex = bson.D{
	{idKey, 1},
}

var eventsQueryIndex = bson.D{
	{aggregateIDKey, 1},
	{versionKey, 1},
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

// GoroutinePoolGoFunc processes actions via provided function
type GoroutinePoolGoFunc = func(func()) error

// EventStore implements an EventStore for MongoDB.
type EventStore struct {
	client          *mongo.Client
	goroutinePoolGo GoroutinePoolGoFunc
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

func New(ctx context.Context, config ConfigV2, logger *zap.Logger, goroutinePoolGo GoroutinePoolGoFunc, opts ...OptionV2) (*EventStore, error) {
	config.marshalerFunc = utils.Marshal
	config.unmarshalerFunc = utils.Unmarshal
	for _, o := range opts {
		o.applyOnV2(&config)
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

	store, err := newEventStoreWithClient(ctx, client, config.Database, "events", config.BatchSize, goroutinePoolGo, config.marshalerFunc, config.unmarshalerFunc, nil)
	if err != nil {
		return nil, err
	}
	store.AddCloseFunc(certManager.Close)
	return store, nil
}

//NewEventStore create a event store from configuration
func NewEventStore(ctx context.Context, config Config, goroutinePoolGo GoroutinePoolGoFunc, opts ...Option) (*EventStore, error) {
	config.marshalerFunc = utils.Marshal
	config.unmarshalerFunc = utils.Unmarshal
	for _, o := range opts {
		o.applyOn(&config)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI).SetMaxPoolSize(config.MaxPoolSize).SetMaxConnIdleTime(config.MaxConnIdleTime).SetTLSConfig(config.tlsCfg))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}

	return newEventStoreWithClient(ctx, client, config.DatabaseName, "events", config.BatchSize, goroutinePoolGo, config.marshalerFunc, config.unmarshalerFunc, nil)
}

// NewEventStore creates a new EventStore.
func newEventStore(ctx context.Context, host, dbPrefix string, colPrefix string, batchSize int, goroutinePoolGo GoroutinePoolGoFunc, eventMarshaler MarshalerFunc, eventUnmarshaler UnmarshalerFunc, LogDebugfFunc LogDebugfFunc, opts ...*options.ClientOptions) (*EventStore, error) {
	newOpts := []*options.ClientOptions{options.Client().ApplyURI("mongodb://" + host)}
	newOpts = append(newOpts, opts...)
	client, err := mongo.Connect(ctx, newOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}

	return newEventStoreWithClient(ctx, client, dbPrefix, colPrefix, batchSize, goroutinePoolGo, eventMarshaler, eventUnmarshaler, LogDebugfFunc)
}

// NewEventStoreWithClient creates a new EventStore with a session.
func newEventStoreWithClient(ctx context.Context, client *mongo.Client, dbPrefix string, colPrefix string, batchSize int, goroutinePoolGo GoroutinePoolGoFunc, eventMarshaler MarshalerFunc, eventUnmarshaler UnmarshalerFunc, LogDebugfFunc LogDebugfFunc) (*EventStore, error) {
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
		goroutinePoolGo: goroutinePoolGo,
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

	return s, nil
}

// IsDup check it error is duplicate
func IsDup(err error) bool {
	// Besides being handy, helps with MongoDB bugs SERVER-7164 and SERVER-11493.
	// What follows makes me sad. Hopefully conventions will be more clear over time.
	switch e := err.(type) {
	case mongo.CommandError:
		return e.Code == 11000 || e.Code == 11001 || e.Code == 12582 || e.Code == 16460 && strings.Contains(e.Message, " E11000 ")
	case mongo.WriteError:
		return e.Code == 11000 || e.Code == 11001 || e.Code == 12582
	case mongo.WriteException:
		isDup := true
		for _, werr := range e.WriteErrors {
			if !IsDup(werr) {
				isDup = false
			}
		}
		return isDup
	}
	return false
}

func (s *EventStore) saveEvent(ctx context.Context, col *mongo.Collection, collectionID string, aggregateID string, event eventstore.Event) (concurrencyException bool, err error) {
	e, err := makeDBEvent(aggregateID, event, s.dataMarshaler)
	if err != nil {
		return false, err
	}

	if _, err := col.InsertOne(ctx, e); err != nil {
		if IsDup(err) {
			return true, nil
		}
		return false, fmt.Errorf("cannot save events: %w", err)
	}
	return false, nil
}

func (s *EventStore) saveEvents(ctx context.Context, col *mongo.Collection, collectionID, aggregateID string, events []eventstore.Event) (concurrencyException bool, err error) {
	firstEvent := true
	version := events[0].Version()
	ops := make([]interface{}, 0, len(events))
	for _, event := range events {
		if firstEvent {
			firstEvent = false
		} else {
			// Only accept events that apply to the correct aggregate version.
			if event.Version() != version+1 {
				return false, errors.New("cannot append unordered events")
			}
			version++
		}

		// Create the event record for the DB.
		e, err := makeDBEvent(aggregateID, event, s.dataMarshaler)
		if err != nil {
			return false, err
		}
		ops = append(ops, e)
	}

	if _, err := col.InsertMany(ctx, ops); err != nil {
		if IsDup(err) {
			return true, nil
		}
		return false, fmt.Errorf("cannot save events: %w", err)
	}
	return false, err
}

type index struct {
	Key  map[string]int
	NS   string
	Name string
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

func getEventCollectionName(groupID string) string {
	return eventCName + "_" + groupID
}

// Save save events to path.
func (s *EventStore) Save(ctx context.Context, collectionID, aggregateID string, events []eventstore.Event) (concurrencyException bool, err error) {
	s.LogDebugfFunc("mongodb.Evenstore.Save start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.Save takes %v", time.Since(t))
	}()

	if len(events) == 0 {
		return false, errors.New("cannot save empty events")
	}
	if aggregateID == "" {
		return false, errors.New("cannot save events without AggregateId")
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName(collectionID))
	if events[0].Version() == 0 {
		err = s.ensureIndex(ctx, col, eventsQueryIndex)
		if err != nil {
			return false, fmt.Errorf("cannot save events: %w", err)
		}
	}

	if events[0].Version() == 0 {
		concurrencyException, err = s.SaveSnapshotQuery(ctx, collectionID, aggregateID, 0)
		if err != nil {
			return false, fmt.Errorf("cannot save events without snapshot query for version 0: %w", err)
		}
		if concurrencyException {
			return concurrencyException, nil
		}
	}

	if len(events) > 1 {
		return s.saveEvents(ctx, col, collectionID, aggregateID, events)
	}
	return s.saveEvent(ctx, col, collectionID, aggregateID, events[0])
}

func (s *EventStore) SaveSnapshot(ctx context.Context, collectionID string, aggregateID string, ev eventstore.Event) (concurrencyException bool, err error) {
	concurrencyException, err = s.Save(ctx, collectionID, aggregateID, []eventstore.Event{ev})
	if !concurrencyException && err == nil {
		return s.SaveSnapshotQuery(ctx, collectionID, aggregateID, ev.Version())
	}
	return concurrencyException, err
}

type iterator struct {
	iter            *mongo.Cursor
	dataUnmarshaler UnmarshalerFunc
	LogDebugfFunc   LogDebugfFunc
	groupID         string
	loaded          bool
}

func (i *iterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	var event bson.M

	if !i.iter.Next(ctx) {
		return nil, false
	}
	i.loaded = true

	err := i.iter.Decode(&event)
	if err != nil {
		return nil, false
	}

	version := event[versionKey].(int64)
	i.LogDebugfFunc("mongodb.iterator.next: GroupId %v: AggregateId %v: Version %v, EvenType %v", i.groupID, event[aggregateIDStrKey].(string), version, event[eventTypeKey].(string))

	data := event[dataKey].(primitive.Binary)
	return eventstore.NewLoadedEvent(
		uint64(version),
		event[eventTypeKey].(string),
		event[aggregateIDStrKey].(string),
		i.groupID,
		func(v interface{}) error {
			return i.dataUnmarshaler(data.Data, v)
		}), true
}

func (i *iterator) Err() error {
	err := i.iter.Err()
	if err == nil && !i.loaded {
		err = fmt.Errorf("none event was loaded")
	}
	return err
}

func versionQueriesToMgoQuery(queries []eventstore.VersionQuery, op signOperator) (bson.M, error) {
	orQueries := make([]bson.D, 0, 32)
	if len(queries) == 0 {
		return bson.M{}, fmt.Errorf("empty []eventstore.VersionQuery")
	}

	for _, q := range queries {
		if q.GroupID == "" {
			return bson.M{}, fmt.Errorf("invalid VersionQuery.GroupID")
		}
		if q.AggregateID == "" {
			return bson.M{}, fmt.Errorf("invalid VersionQuery.AggregateID")
		}
		orQueries = append(orQueries, versionQueryToMgoQuery(q, op))
	}

	return bson.M{"$or": orQueries}, nil
}

func versionQueryToMgoQuery(query eventstore.VersionQuery, op signOperator) bson.D {
	return bson.D{
		{Key: aggregateIDKey, Value: aggregateID2Hash(query.AggregateID)},
		{Key: versionKey, Value: bson.M{string(op): query.Version}},
	}
}

type loader struct {
	store        *EventStore
	eventHandler eventstore.Handler
}

func (l *loader) QueryHandle(ctx context.Context, iter *queryIterator) error {
	var query eventstore.VersionQuery
	queries := make([]eventstore.VersionQuery, 0, 128)
	var errors []error

	for iter.Next(ctx, &query) {
		queries = append(queries, query)
		if len(queries) >= l.store.batchSize {
			err := l.store.LoadFromVersion(ctx, queries, l.eventHandler)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot load events to eventstore model: %w", err))
			}
			queries = queries[:0]
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("loader cannot load events: %v", errors)
	}

	if iter.Err() != nil {
		return iter.Err()
	}

	if len(queries) > 0 {
		return l.store.LoadFromVersion(ctx, queries, l.eventHandler)
	}

	return nil
}

func (l *loader) QueryHandlePool(ctx context.Context, iter *queryIterator) error {
	var query eventstore.VersionQuery
	queries := make([]eventstore.VersionQuery, 0, 128)
	var wg sync.WaitGroup

	var errors []error
	var errorsLock sync.Mutex

	var loaded bool
	for {
		if !iter.Next(ctx, &query) {
			break
		}
		loaded = true
		queries = append(queries, query)
		if len(queries) >= l.store.batchSize {
			wg.Add(1)
			l.store.LogDebugfFunc("mongodb:loader:QueryHandlePool:newTask")
			tmp := queries
			err := l.store.goroutinePoolGo(func() {
				defer wg.Done()
				l.store.LogDebugfFunc("mongodb:loader:QueryHandlePool:task:LoadFromVersion:start")
				err := l.store.LoadFromVersion(ctx, tmp, l.eventHandler)
				l.store.LogDebugfFunc("mongodb:loader:QueryHandlePool:task:LoadFromVersion:done")
				if err != nil {
					errorsLock.Lock()
					defer errorsLock.Unlock()
					errors = append(errors, fmt.Errorf("cannot load events to eventstore model: %w", err))
				}
				l.store.LogDebugfFunc("mongodb:loader:QueryHandlePool:doneTask")
			})
			if err != nil {
				wg.Done()
				errorsLock.Lock()
				errors = append(errors, fmt.Errorf("cannot submit task to load events to eventstore model: %w", err))
				errorsLock.Unlock()
				break
			}
			queries = make([]eventstore.VersionQuery, 0, 128)
		}
	}
	wg.Wait()
	if len(errors) > 0 {
		return fmt.Errorf("loader cannot load events: %v", errors)
	}
	if !loaded {
		return fmt.Errorf("none snapshot event was loaded")
	}

	if iter.Err() != nil {
		return iter.Err()
	}
	if len(queries) > 0 {
		return l.store.LoadFromVersion(ctx, queries, l.eventHandler)
	}

	return nil
}

func (s *EventStore) loadEvents(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler, funcToMgoQuery func(queries []eventstore.VersionQuery) (primitive.M, error)) error {
	collections := make(map[string][]eventstore.VersionQuery)
	for _, query := range queries {
		collections[query.GroupID] = append(collections[query.GroupID], query)
	}

	var errors []error
	for groupID, queries := range collections {
		q, err := funcToMgoQuery(queries)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot load events version: %w", err))
			continue
		}
		err = s.loadMgoQuery(ctx, groupID, eh, q)
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

// LoadUpToVersion loads aggragates events up to a specific version.
func (s *EventStore) LoadUpToVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Eventstore.LoadUpToVersion start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Eventstore.LoadUpToVersion takes %v", time.Since(t))
	}()

	return s.loadEvents(ctx, queries, eh, func(queries []eventstore.VersionQuery) (primitive.M, error) {
		return versionQueriesToMgoQuery(queries, signOperator_lt)
	})
}

// LoadFromVersion loads aggragates events from version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Evenstore.LoadFromVersion start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.LoadFromVersion takes %v", time.Since(t))
	}()
	return s.loadEvents(ctx, queries, eh, func(queries []eventstore.VersionQuery) (primitive.M, error) {
		return versionQueriesToMgoQuery(queries, signOperator_gte)
	})
}

func (s *EventStore) loadMgoQuery(ctx context.Context, groupID string, eh eventstore.Handler, mgoQuery bson.M) error {
	opts := options.FindOptions{}
	opts.SetHint(eventsQueryIndex)
	iter, err := s.client.Database(s.DBName()).Collection(getEventCollectionName(groupID)).Find(ctx, mgoQuery, &opts)
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}

	i := iterator{
		iter:            iter,
		dataUnmarshaler: s.dataUnmarshaler,
		LogDebugfFunc:   s.LogDebugfFunc,
		groupID:         groupID,
	}
	err = eh.Handle(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

// LoadFromSnapshot loads events from the last snapshot eventstore.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot takes %v", time.Since(t))
	}()
	qh := &loader{
		store:        s,
		eventHandler: eventHandler,
	}
	if len(queries) == 0 {
		return fmt.Errorf("not supported")
	}

	collections := make(map[string][]eventstore.SnapshotQuery)
	for _, query := range queries {
		if query.GroupID == "" {
			continue
		}
		if query.AggregateID == "" {
			collections[query.GroupID] = make([]eventstore.SnapshotQuery, 0, 1)
			continue
		}
		v, ok := collections[query.GroupID]
		if !ok {
			v = make([]eventstore.SnapshotQuery, 0, 4)
		} else if len(v) == 0 {
			continue
		}
		v = append(v, query)
		collections[query.GroupID] = v
	}

	var errors []error
	for groupID, queries := range collections {
		err := s.loadSnapshotQueries(ctx, groupID, queries, qh)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
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
func makeDBEvent(aggregateID string, event eventstore.Event, marshaler MarshalerFunc) (bson.M, error) {
	// Marshal event data if there is any.
	raw, err := marshaler(event)
	if err != nil {
		return bson.M{}, fmt.Errorf("cannot create db event: %w", err)
	}

	return bson.M{
		aggregateIDKey:    aggregateID2Hash(aggregateID),
		versionKey:        event.Version(),
		aggregateIDStrKey: aggregateID,
		dataKey:           raw,
		eventTypeKey:      event.EventType(),
		idKey:             aggregateID + "." + strconv.FormatUint(event.Version(), 10),
	}, nil
}

// newDBEvent returns a new dbEvent for an eventstore.
func makeDBSnapshot(aggregateID string, version uint64) bson.M {
	return bson.M{
		aggregateIDKey:    aggregateID2Hash("snapshot"),
		versionKey:        -1,
		aggregateIDStrKey: aggregateID,
		dataKey:           version,
		idKey:             getSnapshotID(aggregateID),
	}
}

// SaveSnapshotQuery upserts the snapshot record
func (s *EventStore) SaveSnapshotQuery(ctx context.Context, groupID, aggregateID string, version uint64) (concurrencyException bool, err error) {
	s.LogDebugfFunc("mongodb.Evenstore.SaveSnapshotQuery start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.SaveSnapshotQuery takes %v", time.Since(t))
	}()

	if aggregateID == "" {
		return false, fmt.Errorf("cannot save snapshot query: invalid query.aggregateID")
	}

	sbSnap := makeDBSnapshot(aggregateID, version)
	col := s.client.Database(s.DBName()).Collection(getEventCollectionName(groupID))
	if version == 0 {
		_, err := col.InsertOne(ctx, sbSnap)
		if err != nil && IsDup(err) {
			// someone update store newer snapshot
			return true, nil
		}
		return false, err
	}

	res, err := col.UpdateOne(ctx,
		bson.M{
			idKey: sbSnap[idKey].(string),
			dataKey: bson.M{
				"$lt": sbSnap[dataKey].(uint64),
			},
		},
		bson.M{
			"$set": sbSnap,
		},
	)
	if err != nil {
		if err == mongo.ErrNilDocument || IsDup(err) {
			// someone update store newer snapshot
			return true, nil
		}
		return false, fmt.Errorf("cannot save snapshot query: %w", err)
	}
	if res.ModifiedCount != 1 {
		return false, fmt.Errorf("cannot update snapshot query")
	}
	return false, nil
}

func getSnapshotID(aggregateID string) string {
	return aggregateID + ".s"
}

func snapshotQueriesToMgoQuery(queries []eventstore.SnapshotQuery) (interface{}, *options.FindOptions) {
	if len(queries) == 0 {
		opts := options.FindOptions{}
		opts.SetHint(eventsQueryIndex)
		return bson.D{{Key: aggregateIDKey, Value: aggregateID2Hash("snapshot")}, {Key: versionKey, Value: -1}}, &opts
	}

	if len(queries) == 1 {
		opts := options.FindOptions{}
		opts.SetHint(snapshotsQueryIndex)
		return bson.D{{Key: idKey, Value: getSnapshotID(queries[0].AggregateID)}}, &opts
	}

	orQueries := make([]bson.M, 0, 32)
	for _, q := range queries {
		if q.AggregateID != "" {
			orQueries = append(orQueries, bson.M{idKey: getSnapshotID(q.AggregateID)})
		}
	}
	opts := options.FindOptions{}
	opts.SetHint(snapshotsQueryIndex)
	return bson.M{"$or": orQueries}, &opts
}

type queryIterator struct {
	iter    *mongo.Cursor
	groupID string
}

func (i *queryIterator) Next(ctx context.Context, q *eventstore.VersionQuery) bool {
	var query bson.M

	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&query)
	if err != nil {
		return false
	}

	version := query[dataKey].(int64)
	q.Version = uint64(version)
	q.AggregateID = query[aggregateIDStrKey].(string)
	q.GroupID = i.groupID
	return true
}

func (i *queryIterator) Err() error {
	return i.iter.Err()
}

func (s *EventStore) loadSnapshotQueries(ctx context.Context, groupID string, queries []eventstore.SnapshotQuery, qh *loader) error {
	var err error
	var iter *mongo.Cursor
	query, hint := snapshotQueriesToMgoQuery(queries)
	if hint == nil {
		iter, err = s.client.Database(s.DBName()).Collection(getEventCollectionName(groupID)).Find(ctx, query)
	} else {
		iter, err = s.client.Database(s.DBName()).Collection(getEventCollectionName(groupID)).Find(ctx, query, hint)
	}
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}
	if s.goroutinePoolGo != nil {
		err = qh.QueryHandlePool(ctx, &queryIterator{iter: iter, groupID: groupID})
	} else {
		err = qh.QueryHandle(ctx, &queryIterator{iter: iter, groupID: groupID})
	}
	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return nil
}

// RemoveUpToVersion deletes the aggragates events up to a specific version.
func (s *EventStore) RemoveUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	collections := make(map[string][]eventstore.VersionQuery)
	for _, query := range queries {
		collections[query.GroupID] = append(collections[query.GroupID], query)
	}

	var errors []error
	for groupID, queries := range collections {
		q, err := versionQueriesToMgoQuery(queries, signOperator_lt)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot load events version: %w", err))
			continue
		}
		_, err = s.client.Database(s.DBName()).Collection(getEventCollectionName(groupID)).DeleteMany(ctx, q)
		if err != nil {
			errors = append(errors, err)
			continue
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

func aggregateID2Hash(aggregateID string) int64 {
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	h.Write([]byte(aggregateID))
	return int64(h.Sum64())
}

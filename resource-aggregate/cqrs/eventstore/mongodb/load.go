package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
)

type iterator struct {
	iter            *mongo.Cursor
	dataUnmarshaler UnmarshalerFunc
	logDebugfFunc   LogDebugfFunc
	queryResolver   *queryResolver

	idx         int
	events      bson.A
	groupID     string
	aggregateID string
	err         error
}

func NewIterator(iter *mongo.Cursor, queryResolver *queryResolver, dataUnmarshaler UnmarshalerFunc, logDebugfFunc LogDebugfFunc) *iterator {
	return &iterator{
		queryResolver:   queryResolver,
		iter:            iter,
		dataUnmarshaler: dataUnmarshaler,
		logDebugfFunc:   logDebugfFunc,
	}
}

func (i *iterator) parseDocument() bool {
	var doc bson.M
	i.err = i.iter.Decode(&doc)
	if i.err != nil {
		return false
	}
	var ok bool
	i.events, ok = doc[eventsKey].(bson.A)
	if !ok {
		i.err = fmt.Errorf("invalid data, %v is not an array", eventsKey)
		return false
	}
	if len(i.events) == 0 {
		i.err = fmt.Errorf("invalid data, no events found")
		return false
	}
	i.groupID, ok = doc[groupIDKey].(string)
	if !ok {
		i.err = fmt.Errorf("invalid data, %v is not a string", groupIDKey)
		return false
	}
	i.aggregateID, ok = doc[aggregateIDKey].(string)
	if !ok {
		i.err = fmt.Errorf("invalid data, %v is not a string", aggregateIDKey)
		return false
	}

	return true
}

func (i *iterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	var ev bson.M
	var version int64
	var ok bool
	for {
		if i.idx >= len(i.events) {
			if !i.iter.Next(ctx) {
				return nil, false
			}
			i.idx = 0
			ok = i.parseDocument()
			if !ok {
				return nil, false
			}
		}
		ev, ok = i.events[i.idx].(bson.M)
		if !ok {
			i.err = fmt.Errorf("invalid data, event %v is not a BSON document", i.idx)
			return nil, false
		}
		version, ok = ev[versionKey].(int64)
		if !ok {
			i.err = fmt.Errorf("invalid data, '%v' of event %v is not an int64", versionKey, i.idx)
			return nil, false
		}
		if i.queryResolver == nil {
			break
		}
		if !i.queryResolver.check(i.aggregateID, version) {
			i.idx++
			continue
		}
		break
	}
	eventType, ok := ev[eventTypeKey].(string)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not a string", eventTypeKey, i.idx)
		return nil, false
	}
	isSnapshot, ok := ev[isSnapshotKey].(bool)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not a bool", isSnapshotKey, i.idx)
		return nil, false
	}
	timestamp, ok := ev[timestampKey].(int64)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not an int64", timestampKey, i.idx)
		return nil, false
	}
	i.logDebugfFunc("mongodb.iterator.next: GroupId %v: AggregateId %v: Version %v, EvenType %v, Timestamp %v",
		i.groupID, i.aggregateID, version, eventType, timestamp)
	data, ok := ev[dataKey].(primitive.Binary)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not a BSON binary data", dataKey, i.idx)
		return nil, false
	}
	i.idx++
	return eventstore.NewLoadedEvent(
		uint64(version),
		eventType,
		i.aggregateID,
		i.groupID,
		isSnapshot,
		pkgTime.Unix(0, timestamp),
		func(v interface{}) error {
			return i.dataUnmarshaler(data.Data, v)
		}), true
}

func (i *iterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return i.iter.Err()
}

type queryResolver struct {
	op                signOperator
	aggregateVersions map[string]uint64
}

func newQueryResolver(op signOperator) *queryResolver {
	return &queryResolver{
		op:                op,
		aggregateVersions: make(map[string]uint64),
	}
}

func (r *queryResolver) set(query eventstore.VersionQuery) error {
	if query.GroupID == "" {
		return fmt.Errorf("invalid GroupID('%v')", query.GroupID)
	}
	if query.AggregateID == "" {
		return fmt.Errorf("invalid AggregateID('%v')", query.AggregateID)
	}
	r.aggregateVersions[query.AggregateID] = query.Version
	return nil
}

func (r *queryResolver) check(aggregateID string, version int64) bool {
	v, ok := r.aggregateVersions[aggregateID]
	if !ok {
		return false
	}
	switch r.op {
	case signOperator_lt:
		return uint64(version) < v
	case signOperator_gte:
		return uint64(version) >= v
	}
	return false
}

/// Create mongodb find query to load events
func (s *EventStore) loadEventsQuery(ctx context.Context, eh eventstore.Handler, queryResolver *queryResolver, filter interface{}, opts ...*options.FindOptions) error {
	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())
	iter, err := col.Find(ctx, filter, opts...)
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}
	i := NewIterator(iter, queryResolver, s.dataUnmarshaler, s.LogDebugfFunc)
	err = eh.Handle(ctx, i)
	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

func (r *queryResolver) toMongoQuery(maxVersionKey string) (filter bson.M, hint bson.D) {
	orQueries := make([]bson.D, 0, 32)
	if len(r.aggregateVersions) == 0 {
		return nil, nil
	}

	for aggregateID, version := range r.aggregateVersions {
		orQueries = append(orQueries, versionQueryToMongoQuery(eventstore.VersionQuery{AggregateID: aggregateID, Version: version}, r.op, maxVersionKey))
	}
	switch maxVersionKey {
	case firstVersionKey:
		hint = aggregateIDFirstVersionQueryIndex
	case latestVersionKey:
		hint = aggregateIDLastVersionQueryIndex
	}

	return bson.M{"$or": orQueries}, hint
}

func versionQueryToMongoQuery(query eventstore.VersionQuery, op signOperator, maxVersionKey string) bson.D {
	if op == signOperator_lt {
		return bson.D{
			{Key: aggregateIDKey, Value: query.AggregateID},
			{Key: maxVersionKey, Value: bson.M{string(op): query.Version}},
		}
	}
	return bson.D{
		{Key: aggregateIDKey, Value: query.AggregateID},
		{Key: maxVersionKey, Value: bson.M{string(op): query.Version}},
	}
}

// LoadUpToVersion loads aggregates events up to a specific version.
func (s *EventStore) LoadUpToVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	return s.loadEvents(ctx, queries, eh, signOperator_lt, firstVersionKey)
}

func (s *EventStore) loadEvents(ctx context.Context, versionQueries []eventstore.VersionQuery, eh eventstore.Handler, op signOperator, maxVersionKey string) error {
	normalizedVersionQueries := make(map[string][]eventstore.VersionQuery)
	for _, query := range versionQueries {
		normalizedVersionQueries[query.GroupID] = append(normalizedVersionQueries[query.GroupID], query)
	}

	var errors []error
	for _, queries := range normalizedVersionQueries {
		queryResolver := newQueryResolver(op)
		for _, q := range queries {
			err := queryResolver.set(q)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot load events version for query('%+v'): %w", q, err))
				continue
			}
		}
		err := s.loadMongoQuery(ctx, eh, queryResolver, maxVersionKey)
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

// LoadFromVersion loads aggregates events from version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	return s.loadEvents(ctx, queries, eh, signOperator_gte, latestVersionKey)
}

func (s *EventStore) loadMongoQuery(ctx context.Context, eh eventstore.Handler, queryResolver *queryResolver, maxVersionKey string) error {
	filter, hint := queryResolver.toMongoQuery(maxVersionKey)
	opts := options.Find()
	opts.SetHint(hint)
	return s.loadEventsQuery(ctx, eh, queryResolver, filter, opts)
}

func snapshotQueriesToMongoQuery(groupID string, queries []eventstore.SnapshotQuery) (interface{}, *options.FindOptions) {
	opts := options.Find()
	opts.SetAllowDiskUse(true)
	opts.SetProjection(bson.M{
		groupIDKey:     1,
		aggregateIDKey: 1,
		eventsKey: bson.M{
			"$filter": bson.M{
				"input": "$" + eventsKey,
				"as":    eventsKey,
				"cond": bson.M{
					string(signOperator_gte): []string{"$$" + eventsKey + "." + versionKey, "$" + latestSnapshotVersionKey},
				},
			},
		},
	})
	if len(queries) == 0 {
		opts.SetHint(groupIDQueryIndex)
		return bson.D{
			{Key: groupIDKey, Value: groupID}, {Key: isActiveKey, Value: true},
		}, opts
	}

	opts.SetHint(groupIDaggregateIDQueryIndex)
	orQueries := make([]bson.D, 0, 32)
	for _, q := range queries {
		if q.AggregateID != "" {
			orQueries = append(orQueries, bson.D{{Key: groupIDKey, Value: groupID}, {Key: aggregateIDKey, Value: q.AggregateID}, {Key: isActiveKey, Value: true}})
		}
	}
	return bson.M{
		"$or": orQueries,
	}, opts
}

func (s *EventStore) loadFromSnapshot(ctx context.Context, groupID string, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	filter, opts := snapshotQueriesToMongoQuery(groupID, queries)
	return s.loadEventsQuery(ctx, eventHandler, nil, filter, opts)
}

// LoadFromSnapshot loads events from the last snapshot eventstore.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot takes %v", time.Since(t))
	}()
	if len(queries) == 0 {
		return fmt.Errorf("not supported")
	}

	normalizeQuery := make(map[string][]eventstore.SnapshotQuery)
	for _, query := range queries {
		if query.GroupID == "" {
			continue
		}
		if query.AggregateID == "" {
			normalizeQuery[query.GroupID] = make([]eventstore.SnapshotQuery, 0, 1)
			continue
		}
		v, ok := normalizeQuery[query.GroupID]
		if !ok {
			v = make([]eventstore.SnapshotQuery, 0, 4)
		} else if len(v) == 0 {
			continue
		}
		v = append(v, query)
		normalizeQuery[query.GroupID] = v
	}

	var errors []error
	for groupID, queries := range normalizeQuery {
		err := s.loadFromSnapshot(ctx, groupID, queries, eventHandler)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

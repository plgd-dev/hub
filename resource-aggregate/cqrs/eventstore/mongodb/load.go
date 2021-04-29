package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

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
}

func NewIterator(iter *mongo.Cursor, queryResolver *queryResolver, dataUnmarshaler UnmarshalerFunc, logDebugfFunc LogDebugfFunc) *iterator {
	return &iterator{
		queryResolver:   queryResolver,
		iter:            iter,
		dataUnmarshaler: dataUnmarshaler,
		logDebugfFunc:   logDebugfFunc,
	}
}

func (i *iterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	for {
		if i.idx >= len(i.events) {
			if !i.iter.Next(ctx) {
				return nil, false
			}
			i.idx = 0
			var doc bson.M
			err := i.iter.Decode(&doc)
			if err != nil {
				return nil, false
			}
			i.events = doc[eventsKey].(bson.A)
			if len(i.events) == 0 {
				return nil, false
			}
			i.groupID = doc[groupIDKey].(string)
			i.aggregateID = doc[aggregateIDKey].(string)
		}
		ev := i.events[i.idx].(bson.M)
		version := ev[versionKey].(int64)
		if i.queryResolver == nil {
			break
		}
		if !i.queryResolver.check(i.aggregateID, version) {
			i.idx++
			continue
		}
		break
	}
	ev := i.events[i.idx].(bson.M)
	i.idx++
	version := ev[versionKey].(int64)
	eventType := ev[eventTypeKey].(string)
	isSnapshot := ev[isSnapshotKey].(bool)
	i.logDebugfFunc("mongodb.iterator.next: GroupId %v: AggregateId %v: Version %v, EvenType %v", i.groupID, i.aggregateID, version, eventType)
	data := ev[dataKey].(primitive.Binary)
	return eventstore.NewLoadedEvent(
		uint64(version),
		eventType,
		i.aggregateID,
		i.groupID,
		isSnapshot,
		func(v interface{}) error {
			return i.dataUnmarshaler(data.Data, v)
		}), true
}

func (i *iterator) Err() error {
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

// LoadUpToVersion loads aggragates events up to a specific version.
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

// LoadFromVersion loads aggragates events from version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	return s.loadEvents(ctx, queries, eh, signOperator_gte, latestVersionKey)
}

func (s *EventStore) loadMongoQuery(ctx context.Context, eh eventstore.Handler, queryResolver *queryResolver, maxVersionKey string) error {
	filter, hint := queryResolver.toMongoQuery(maxVersionKey)
	opts := options.Find()
	opts.SetHint(hint)
	iter, err := s.client.Database(s.DBName()).Collection(getEventCollectionName()).Find(ctx, filter, opts)
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

func snapshotQueriesToMongoQuery(groupID string, queries []eventstore.SnapshotQuery) (mongo.Pipeline, *options.AggregateOptions) {
	opts := options.Aggregate()
	opts.SetAllowDiskUse(true)
	if len(queries) == 0 {
		opts.SetHint(groupIDQueryIndex)
		return mongo.Pipeline{
			bson.D{
				{
					Key: "$match",
					Value: bson.D{
						{Key: groupIDKey, Value: groupID}, {Key: isActiveKey, Value: true},
					},
				},
			},
			bson.D{
				{
					Key: "$project",
					Value: bson.M{
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
					},
				},
			}}, opts
	}

	opts.SetHint(groupIDaggregateIDQueryIndex)
	orQueries := make([]bson.D, 0, 32)
	for _, q := range queries {
		if q.AggregateID != "" {
			orQueries = append(orQueries, bson.D{{Key: groupIDKey, Value: groupID}, {Key: aggregateIDKey, Value: q.AggregateID}, {Key: isActiveKey, Value: true}})
		}
	}
	return mongo.Pipeline{
		bson.D{
			{
				Key: "$match",
				Value: bson.M{
					"$or": orQueries,
				},
			},
		},
		bson.D{
			{
				Key: "$project",
				Value: bson.M{
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
				},
			},
		},
	}, opts
}

func (s *EventStore) loadFromSnapshot(ctx context.Context, groupID string, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	var err error
	var iter *mongo.Cursor
	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())
	pipeline, opts := snapshotQueriesToMongoQuery(groupID, queries)
	iter, err = col.Aggregate(ctx, pipeline, opts)
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}
	i := NewIterator(iter, nil, s.dataUnmarshaler, s.LogDebugfFunc)
	err = eventHandler.Handle(ctx, i)
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

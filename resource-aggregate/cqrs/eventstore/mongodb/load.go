package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/internal/math"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

const errFmtDataIsNotStringType = "invalid data['%v'] type ('%T'), expected string type"

func newIterator(iter *mongo.Cursor, queryResolver *queryResolver, dataUnmarshaler UnmarshalerFunc, logDebugfFunc LogDebugfFunc) *iterator {
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
		i.err = errors.New("invalid data, no events found")
		return false
	}
	i.groupID, ok = doc[groupIDKey].(string)
	if !ok {
		i.err = fmt.Errorf(errFmtDataIsNotStringType, groupIDKey, doc[groupIDKey])
		return false
	}
	i.aggregateID, ok = doc[aggregateIDKey].(string)
	if !ok {
		i.err = fmt.Errorf(errFmtDataIsNotStringType, aggregateIDKey, doc[aggregateIDKey])
		return false
	}

	return true
}

func (i *iterator) getEvent() (bson.M, uint64, error) {
	ev, ok := i.events[i.idx].(bson.M)
	if !ok {
		return nil, 0, fmt.Errorf("invalid data, event %v is not a BSON document", i.idx)
	}
	version, ok := ev[versionKey].(int64)
	if !ok {
		return nil, 0, fmt.Errorf("invalid data, '%v' of event %v is not an int64", versionKey, i.idx)
	}
	return ev, math.CastTo[uint64](version), nil
}

func (i *iterator) nextEvent(ctx context.Context) (bson.M, uint64) {
	var ev bson.M
	var version uint64
	var ok bool
	for {
		if i.idx >= len(i.events) {
			if !i.iter.Next(ctx) {
				return nil, 0
			}
			i.idx = 0
			ok = i.parseDocument()
			if !ok {
				return nil, 0
			}
		}
		var err error
		ev, version, err = i.getEvent()
		if err != nil {
			i.err = err
			return nil, 0
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
	return ev, version
}

func (i *iterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	ev, version := i.nextEvent(ctx)
	if ev == nil {
		return nil, false
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
	i.logDebugfFunc("mongodb.iterator.next: GroupID %v: AggregateID %v: Version %v, EvenType %v, Timestamp %v",
		i.groupID, i.aggregateID, version, eventType, timestamp)
	data, ok := ev[dataKey].(primitive.Binary)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not a BSON binary data", dataKey, i.idx)
		return nil, false
	}
	i.idx++
	return eventstore.NewLoadedEvent(
		version,
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
	if query.AggregateID == "" || query.AggregateID == uuid.Nil.String() {
		return fmt.Errorf("invalid AggregateID('%v')", query.AggregateID)
	}
	r.aggregateVersions[query.AggregateID] = query.Version
	return nil
}

func (r *queryResolver) check(aggregateID string, version uint64) bool {
	v, ok := r.aggregateVersions[aggregateID]
	if !ok {
		return false
	}
	switch r.op {
	case signOperator_lt:
		return version < v
	case signOperator_gte:
		return version >= v
	}
	return false
}

// Create mongodb find query to load events
func (s *EventStore) loadEventsQuery(ctx context.Context, eh eventstore.Handler, queryResolver *queryResolver, queries []mongoQuery) error {
	col := s.client().Database(s.DBName()).Collection(getEventCollectionName())
	var errs *multierror.Error
	for _, q := range queries {
		iter, err := col.Find(ctx, q.filter, q.options)
		if errors.Is(err, mongo.ErrNilDocument) {
			continue
		}
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		i := newIterator(iter, queryResolver, s.dataUnmarshaler, s.LogDebugfFunc)
		err = eh.Handle(ctx, i)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		err = iter.Close(ctx)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
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

	var errors *multierror.Error
	for _, queries := range normalizedVersionQueries {
		queryResolver := newQueryResolver(op)
		for _, q := range queries {
			err := queryResolver.set(q)
			if err != nil {
				errors = multierror.Append(errors, fmt.Errorf("cannot load events version for query('%+v'): %w", q, err))
				continue
			}
		}
		err := s.loadMongoQuery(ctx, eh, queryResolver, maxVersionKey)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
	}
	return errors.ErrorOrNil()
}

// LoadFromVersion loads aggregates events from version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	return s.loadEvents(ctx, queries, eh, signOperator_gte, latestVersionKey)
}

func (s *EventStore) loadMongoQuery(ctx context.Context, eh eventstore.Handler, queryResolver *queryResolver, maxVersionKey string) error {
	filter, hint := queryResolver.toMongoQuery(maxVersionKey)
	opts := options.Find()
	opts.SetHint(hint)
	return s.loadEventsQuery(ctx, eh, queryResolver, []mongoQuery{{filter: filter, options: opts}})
}

type mongoQuery struct {
	filter  interface{}
	options *options.FindOptions
}

func snapshotQueriesToMongoQuery(groupID string, queries []eventstore.SnapshotQuery) []mongoQuery {
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
		return []mongoQuery{
			{
				filter: bson.D{
					{Key: groupIDKey, Value: groupID}, {Key: isActiveKey, Value: true},
				},
				options: opts,
			},
		}
	}

	optsTypes := options.Find()
	// create a copy of the options
	*optsTypes = *opts
	resourceQueries := make([]bson.D, 0, 32)
	typeQueries := make([]bson.D, 0, 32)
	for _, q := range queries {
		if (q.AggregateID == "" || q.AggregateID == uuid.Nil.String()) && len(q.Types) == 0 {
			opts.SetHint(groupIDQueryIndex)
			return []mongoQuery{
				{
					filter: bson.D{
						{Key: groupIDKey, Value: groupID}, {Key: isActiveKey, Value: true},
					},
					options: opts,
				},
			}
		}
		if (q.AggregateID == "" || q.AggregateID == uuid.Nil.String()) && len(q.Types) > 0 {
			optsTypes.SetHint(groupIDTypesQueryIndex)
			typeQueries = append(typeQueries, bson.D{{Key: groupIDKey, Value: groupID}, {Key: typesKey, Value: bson.M{"$all": q.Types}}, {Key: isActiveKey, Value: true}})
			continue
		}

		if q.AggregateID != "" && q.AggregateID != uuid.Nil.String() {
			opts.SetHint(groupIDaggregateIDQueryIndex)
			resourceQueries = append(resourceQueries, bson.D{{Key: groupIDKey, Value: groupID}, {Key: aggregateIDKey, Value: q.AggregateID}, {Key: isActiveKey, Value: true}})
		}
	}

	r := make([]mongoQuery, 0, 2)
	if len(resourceQueries) > 0 {
		r = append(r, mongoQuery{
			filter:  bson.M{"$or": resourceQueries},
			options: opts,
		})
	}
	if len(typeQueries) > 0 {
		r = append(r, mongoQuery{
			filter:  bson.M{"$or": typeQueries},
			options: optsTypes,
		})
	}
	return r
}

func (s *EventStore) loadFromSnapshot(ctx context.Context, groupID string, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	mongoQueries := snapshotQueriesToMongoQuery(groupID, queries)
	if len(mongoQueries) > 1 {
		return errors.New("too many types of queries")
	}
	return s.loadEventsQuery(ctx, eventHandler, nil, mongoQueries)
}

func resourceTypesIsSubset(slice, subset []string) bool {
	if len(slice) == 0 {
		return false
	}
	if len(slice) > len(subset) {
		return false
	}
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	for _, s := range subset {
		delete(set, s)
		if len(set) == 0 {
			return true
		}
	}
	return false
}

func uniqueQueryWithEmptyAggregateID(queries []eventstore.SnapshotQuery, query eventstore.SnapshotQuery) []eventstore.SnapshotQuery {
	if len(query.Types) == 0 {
		// get all events without filter
		return []eventstore.SnapshotQuery{query}
	}
	for idx, q := range queries {
		if resourceTypesIsSubset(query.Types, q.Types) {
			// get all events with the certain type, replace the queries with the more or equal general one
			queries[idx] = query
			queries = removeDuplicates(queries, idx, func(qa eventstore.SnapshotQuery) bool { return resourceTypesIsSubset(query.Types, qa.Types) })
			return queries
		}
	}
	return append(queries, query)
}

func uniqueQueryHandleAtIndex(queries []eventstore.SnapshotQuery, query eventstore.SnapshotQuery, idx int) ([]eventstore.SnapshotQuery, bool) {
	q := queries[idx]
	if (q.AggregateID == "" || q.AggregateID == uuid.Nil.String()) && len(q.Types) == 0 || resourceTypesIsSubset(q.Types, query.Types) {
		return queries, true // No need to add more specific one if there's a general query
	}

	if q.AggregateID == query.AggregateID {
		if len(query.Types) == 0 {
			queries[idx] = query
			queries = removeDuplicates(queries, idx, func(qa eventstore.SnapshotQuery) bool { return qa.AggregateID == query.AggregateID })
			return queries, true
		}
		if len(q.Types) == 0 || resourceTypesIsSubset(q.Types, query.Types) {
			return queries, true // No need to add more general one if there's a more specific query
		}
		if resourceTypesIsSubset(query.Types, q.Types) {
			// replace query with the more general one
			queries[idx] = query
			queries = removeDuplicates(queries, idx, func(qa eventstore.SnapshotQuery) bool { return resourceTypesIsSubset(qa.Types, query.Types) })
			return queries, true
		}
	}
	return queries, false
}

func uniqueQuery(queries []eventstore.SnapshotQuery, query eventstore.SnapshotQuery) []eventstore.SnapshotQuery {
	if query.AggregateID == "" || query.AggregateID == uuid.Nil.String() {
		return uniqueQueryWithEmptyAggregateID(queries, query)
	}

	for idx := range queries {
		newQueries, handled := uniqueQueryHandleAtIndex(queries, query, idx)
		if handled {
			return newQueries
		}
	}

	return append(queries, query)
}

func removeDuplicates(queries []eventstore.SnapshotQuery, startIdx int, remove func(q eventstore.SnapshotQuery) bool) []eventstore.SnapshotQuery {
	for i := startIdx + 1; i < len(queries); i++ {
		if remove(queries[i]) {
			queries = append(queries[:i], queries[i+1:]...)
			i--
		}
	}
	return queries
}

func normalizeSnapshotQuery(queries []eventstore.SnapshotQuery) map[string][]eventstore.SnapshotQuery {
	normalizedQuery := make(map[string][]eventstore.SnapshotQuery, len(queries))
	// split queries by groupID
	for _, query := range queries {
		if query.GroupID == "" {
			continue
		}
		v, ok := normalizedQuery[query.GroupID]
		if !ok {
			v = make([]eventstore.SnapshotQuery, 0, 4)
		}
		normalizedQuery[query.GroupID] = uniqueQuery(v, query)
	}
	return normalizedQuery
}

// LoadFromSnapshot loads events from the last snapshot eventstore.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.LoadFromSnapshot takes %v", time.Since(t))
	}()
	if len(queries) == 0 {
		return errors.New("not supported")
	}

	normalizeQuery := normalizeSnapshotQuery(queries)

	var errors *multierror.Error
	for groupID, queries := range normalizeQuery {
		err := s.loadFromSnapshot(ctx, groupID, queries, eventHandler)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

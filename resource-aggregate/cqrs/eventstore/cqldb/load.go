package cqldb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type iterator struct {
	iter            *gocql.Iter
	dataUnmarshaler UnmarshalerFunc
	err             error
}

const (
	errFmtEventDataIsNotUUIDType = "invalid event['%v'] data['%v'] type ('%T'), expected gocql.UUID"
	errFmtDataIsNotUUIDType      = "invalid data['%v'] type, ('%T'), expected gocql.UUID"
)

func newIterator(iter *gocql.Iter, dataUnmarshaler UnmarshalerFunc) *iterator {
	return &iterator{
		iter:            iter,
		dataUnmarshaler: dataUnmarshaler,
	}
}

func (i *iterator) nextEvent(_ context.Context) map[string]interface{} {
	if i.err != nil {
		return nil
	}
	v := make(map[string]interface{})
	ok := i.iter.MapScan(v)
	if !ok {
		return nil
	}
	return v
}

func (i *iterator) Next(ctx context.Context) (eventstore.EventUnmarshaler, bool) {
	ev := i.nextEvent(ctx)
	if ev == nil {
		return nil, false
	}
	eventType, ok := ev[eventTypeKey].(string)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event is not a string but %T", eventTypeKey, ev[eventTypeKey])
		return nil, false
	}
	isSnapshot := true
	timestamp, ok := ev[timestampKey].(int64)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not an int64 but %T", timestampKey, eventTypeKey, ev[timestampKey])
		return nil, false
	}
	version, ok := ev[versionKey].(int64)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not an int64 but %T", versionKey, eventTypeKey, ev[versionKey])
		return nil, false
	}
	aggregateID, ok := ev[idKey].(gocql.UUID)
	if !ok {
		i.err = fmt.Errorf(errFmtEventDataIsNotUUIDType, eventTypeKey, idKey, ev[idKey])
		return nil, false
	}
	groupID, ok := ev[deviceIDKey].(gocql.UUID)
	if !ok {
		i.err = fmt.Errorf(errFmtEventDataIsNotUUIDType, eventTypeKey, deviceIDKey, ev[deviceIDKey])
		return nil, false
	}
	snapshot, ok := ev[snapshotKey].([]byte)
	if !ok {
		i.err = fmt.Errorf("invalid data, '%v' of event %v is not a []byte but %T", snapshotKey, eventTypeKey, ev[snapshotKey])
		return nil, false
	}
	return eventstore.NewLoadedEvent(
		uint64(version),
		eventType,
		aggregateID.String(),
		groupID.String(),
		isSnapshot,
		pkgTime.Unix(0, timestamp),
		func(v interface{}) error {
			return i.dataUnmarshaler(snapshot, v)
		}), true
}

func (i *iterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return nil
}

// Create cqldb find query to load events
func (s *EventStore) loadEventsQuery(ctx context.Context, eh eventstore.Handler, filter string) error {
	var q strings.Builder
	q.WriteString("select * from " + s.Table())
	if filter != "" {
		q.WriteString(" " + cqldb.WhereClause + " " + filter)
	}
	q.WriteString(";")
	iter := s.Session().Query(q.String()).WithContext(ctx).Iter()
	i := newIterator(iter, s.unmarshalerFunc)
	err := eh.Handle(ctx, i)
	errClose := iter.Close()
	if err == nil {
		return errClose
	}
	return err
}

// LoadUpToVersion loads aggregates events up to a specific version. It loads events from the last snapshot. Only last snapshots
func (s *EventStore) LoadUpToVersion(_ context.Context, _ []eventstore.VersionQuery, _ eventstore.Handler) error {
	return eventstore.ErrNotSupported
}

// LoadFromVersion loads aggregates events from version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eh eventstore.Handler) error {
	q := make([]eventstore.SnapshotQuery, 0, len(queries))
	for _, query := range queries {
		q = append(q, eventstore.SnapshotQuery{
			GroupID:     query.GroupID,
			AggregateID: query.AggregateID,
		})
	}
	return s.LoadFromSnapshot(ctx, q, eh)
}

func addAggregateIDsToFilter(filter *strings.Builder, queries []eventstore.SnapshotQuery) {
	aggrs := make([]string, 0, len(queries))
	for _, q := range queries {
		if q.AggregateID != "" && q.AggregateID != uuid.Nil.String() {
			aggrs = append(aggrs, q.AggregateID)
		}
	}
	if len(aggrs) > 0 {
		if filter.Len() > 0 {
			filter.WriteString(" and ")
		}
		filter.WriteString(idKey)
		filter.WriteString(" in (")
		for idx, aggr := range aggrs {
			if idx > 0 {
				filter.WriteString(",")
			}
			filter.WriteString(aggr)
		}
		filter.WriteString(")")
	}
}

func addTimestampToFilter(filter *strings.Builder, timestamp int64) {
	if timestamp > 0 {
		if filter.Len() > 0 {
			filter.WriteString(" and ")
		}
		filter.WriteString(timestampKey)
		filter.WriteString(">=")
		filter.WriteString(strconv.FormatInt(timestamp, 10))
		filter.WriteString(" ALLOW FILTERING")
	}
}

func snapshotQueriesToFilter(deviceID string, queries []eventstore.SnapshotQuery, timestamp int64) string {
	var filter strings.Builder
	if deviceID != "" {
		filter.WriteString(deviceIDKey + "=" + deviceID)
	}
	addAggregateIDsToFilter(&filter, queries)
	addTimestampToFilter(&filter, timestamp)
	return filter.String()
}

func (s *EventStore) loadFromSnapshotByGroup(ctx context.Context, groupID string, queries []eventstore.SnapshotQuery, timestamp int64, eventHandler eventstore.Handler) error {
	if groupID != "" {
		if _, err := uuid.Parse(groupID); err != nil {
			return fmt.Errorf("invalid groupID %v: %w", groupID, err)
		}
	}
	for _, query := range queries {
		if _, err := uuid.Parse(query.AggregateID); err != nil {
			return fmt.Errorf("invalid aggregateID %v: %w", query.AggregateID, err)
		}
	}

	filter := snapshotQueriesToFilter(groupID, queries, timestamp)
	return s.loadEventsQuery(ctx, eventHandler, filter)
}

func normalizeQueries(queries []eventstore.SnapshotQuery) map[string][]eventstore.SnapshotQuery {
	normalizeQuery := make(map[string][]eventstore.SnapshotQuery, len(queries))
	for _, query := range queries {
		if query.AggregateID == "" || query.AggregateID == uuid.Nil.String() {
			normalizeQuery[query.GroupID] = nil
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
	return normalizeQuery
}

func (s *EventStore) loadFromSnapshot(ctx context.Context, normalizedQueries map[string][]eventstore.SnapshotQuery, timestamp int64, eventHandler eventstore.Handler) error {
	var errors *multierror.Error
	if len(normalizedQueries) == 0 {
		return s.loadFromSnapshotByGroup(ctx, "", nil, timestamp, eventHandler)
	}
	for groupID, queries := range normalizedQueries {
		err := s.loadFromSnapshotByGroup(ctx, groupID, queries, timestamp, eventHandler)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

// LoadFromSnapshot loads events from the last snapshot eventstore.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler eventstore.Handler) error {
	normalizedQueries := normalizeQueries(queries)
	if len(normalizedQueries) == 0 {
		return status.Errorf(codes.InvalidArgument, "invalid queries")
	}
	return s.loadFromSnapshot(ctx, normalizedQueries, 0, eventHandler)
}

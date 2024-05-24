package cqldb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

type keyValue struct {
	Key string
	Val string
}

type keyValues []keyValue

func (k keyValues) Keys() []string {
	keys := make([]string, 0, len(k))
	for _, kv := range k {
		keys = append(keys, kv.Key)
	}
	return keys
}

func (k keyValues) Values() []string {
	values := make([]string, 0, len(k))
	for _, kv := range k {
		values = append(values, kv.Val)
	}
	return values
}

func (k keyValues) String() string {
	var b strings.Builder
	for idx, kv := range k {
		if idx > 0 {
			b.WriteString(",")
		}
		b.WriteString(kv.Key)
		b.WriteString("=")
		b.WriteString(kv.Val)
	}
	return b.String()
}

func eventToKeyValues(event eventstore.Event, insert bool, data []byte) keyValues {
	kvs := make([]keyValue, 0, 16)
	if insert {
		kvs = append(kvs, keyValue{Key: deviceIDKey, Val: event.GroupID()})
		kvs = append(kvs, keyValue{Key: idKey, Val: event.AggregateID()})
		kvs = append(kvs, keyValue{Key: eventTypeKey, Val: "'" + event.EventType() + "'"})
	}
	kvs = append(kvs, keyValue{Key: versionKey, Val: strconv.FormatUint(event.Version(), 10)})
	kvs = append(kvs, keyValue{Key: timestampKey, Val: strconv.FormatInt(event.Timestamp().UnixNano(), 10)})
	kvs = append(kvs, keyValue{Key: snapshotKey, Val: encodeToBlob(data)})
	serviceID, ok := event.ServiceID()
	if ok {
		if serviceID == "" {
			serviceID = "null"
		}
		kvs = append(kvs, keyValue{Key: serviceIDKey, Val: serviceID})
	} else if insert {
		kvs = append(kvs, keyValue{Key: serviceIDKey, Val: "null"})
	}
	etagData := event.ETag()
	if etagData != nil {
		kvs = append(kvs, keyValue{Key: etagKey, Val: encodeToBlob(etagData.ETag)})
		kvs = append(kvs, keyValue{Key: etagTimestampKey, Val: strconv.FormatInt(etagData.Timestamp, 10)})
	} else {
		kvs = append(kvs, keyValue{Key: etagKey, Val: "null"})
		kvs = append(kvs, keyValue{Key: etagTimestampKey, Val: "0"})
	}
	return kvs
}

func eventsToCQLSetValue(event eventstore.Event, data []byte) string {
	kvs := eventToKeyValues(event, false, data)
	return kvs.String()
}

func (s *EventStore) saveEvent(ctx context.Context, events []eventstore.Event) (status eventstore.SaveStatus, err error) {
	lastEvent, snapshotBinary, err := getLatestEventsSnapshot(events, s.marshalerFunc)
	if err != nil {
		return eventstore.Fail, err
	}
	setters := eventsToCQLSetValue(lastEvent, snapshotBinary)
	q := "update " + s.Table() + " set " + setters + " " + cqldb.WhereClause + " " + deviceIDKey + "=" + lastEvent.GroupID() + " and " + idKey + "=" + lastEvent.AggregateID() + " if " + versionKey + "=" + strconv.FormatUint(events[0].Version()-1, 10) + ";"
	ok, err := s.Session().Query(q).WithContext(ctx).ScanCAS(nil)
	if err != nil {
		return eventstore.Fail, fmt.Errorf("cannot update snapshot event('%v'): %w", events, err)
	}
	if !ok {
		return eventstore.ConcurrencyException, nil
	}
	return eventstore.Ok, nil
}

// Save save events to eventstore.
// AggregateID, GroupID and EventType are required.
// All events within one Save operation shall have the same AggregateID and GroupID.
// Versions shall be unique and ascend continually.
// Only first event can be a snapshot.
func (s *EventStore) Save(ctx context.Context, events ...eventstore.Event) (eventstore.SaveStatus, error) {
	if err := eventstore.ValidateEventsBeforeSave(events); err != nil {
		return eventstore.Fail, err
	}
	if events[0].Version() != 0 {
		return s.saveEvent(ctx, events)
	}
	lastEvent, data, err := getLatestEventsSnapshot(events, s.marshalerFunc)
	if err != nil {
		return eventstore.Fail, err
	}
	kvs := eventToKeyValues(lastEvent, true, data)
	keys := kvs.Keys()
	values := kvs.Values()

	q := "insert into " + s.Table() + " (" + strings.Join(keys, ",") + ") values (" + strings.Join(values, ",") + ") if not exists;"
	ok, err := s.Session().Query(q).WithContext(ctx).ScanCAS(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return eventstore.Ok, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot insert first events('%v'): %w", events, err)
	}
	if !ok {
		return eventstore.ConcurrencyException, nil
	}
	return eventstore.Ok, nil
}

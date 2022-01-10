package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

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

func (s *EventStore) saveEvent(ctx context.Context, col *mongo.Collection, events []eventstore.Event) (status eventstore.SaveStatus, err error) {
	e, err := makeDBEvents(events, s.dataMarshaler)
	if err != nil {
		return eventstore.Fail, err
	}
	opts := options.Update()
	opts.SetHint(aggregateIDLastVersionQueryIndex)
	updateSet := bson.M{
		latestVersionKey:   events[len(events)-1].Version(),
		latestTimestampKey: events[len(events)-1].Timestamp().UnixNano(),
	}
	latestSnapshotVersion, err := getLatestSnapshotVersion(events)
	if err == nil {
		updateSet[latestSnapshotVersionKey] = latestSnapshotVersion
	}

	// find document of aggregate with previous version
	// latestVersion shall be lower by 1 as new event otherwise other event was stored (occ).
	filter := bson.D{
		{Key: aggregateIDKey, Value: events[0].AggregateID()},
		{Key: latestVersionKey, Value: events[0].Version() - 1},
	}
	update := bson.M{
		"$set": updateSet,
		"$push": bson.M{
			eventsKey: bson.M{
				"$each": e,
			},
		},
	}

	res, err := col.UpdateOne(ctx, filter, update, opts)
	switch err {
	case nil:
		if res.ModifiedCount == 0 {
			return eventstore.ConcurrencyException, nil
		}
		return eventstore.Ok, nil
	case mongo.ErrNilDocument:
		return eventstore.ConcurrencyException, nil
	default:
		switch wErr := err.(type) {
		case mongo.WriteException:
			var sizeIsExceeded bool
			for _, e := range wErr.WriteErrors {
				if e.Code == 10334 {
					sizeIsExceeded = true
					break
				}
			}
			if !sizeIsExceeded {
				return eventstore.Fail, fmt.Errorf("cannot push events('%v') to db: %w", events, err)
			}
			return eventstore.SnapshotRequired, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot push events('%v') to db: %w", events, err)
	}
}

// Validate events before saving them in the database
// Prerequisites that must hold:
//   1) AggregateID, GroupID and EventType are not empty
//   2) Version for each event is by 1 greater than the version of the previous event
//   3) Only the first event can be a snapshot
//   4) All events have the same AggregateId and GroupID
//   5) Timestamps are non-zero
//   6) Timestamps are non-decreasing
func checkBeforeSave(events ...eventstore.Event) error {
	if len(events) == 0 || events[0] == nil {
		return fmt.Errorf("invalid events('%v')", events)
	}
	aggregateID := events[0].AggregateID()
	groupID := events[0].GroupID()
	version := events[0].Version()
	timestamp := events[0].Timestamp()
	for idx, event := range events {
		if event == nil {
			return fmt.Errorf("invalid events[%v]('%v')", idx, event)
		}
		if event.AggregateID() == "" {
			return fmt.Errorf("invalid events[%v].AggregateID('%v')", idx, event.AggregateID())
		}
		if event.GroupID() == "" {
			return fmt.Errorf("invalid events[%v].GroupID('%v')", idx, event.GroupID())
		}
		if event.EventType() == "" {
			return fmt.Errorf("invalid events[%v].EventType('%v')", idx, event.EventType())
		}
		if event.Timestamp().IsZero() {
			return fmt.Errorf("invalid zero events[%v].Timestamp", idx)
		}
		if idx > 0 {
			if event.Version() != version+uint64(idx) {
				return fmt.Errorf("invalid continues ascending events[%v].Version(%v))", idx, event.Version())
			}
			if event.AggregateID() != aggregateID {
				return fmt.Errorf("invalid events[%v].AggregateID('%v') != events[0].AggregateID('%v')", idx, event.AggregateID(), aggregateID)
			}
			if event.GroupID() != groupID {
				return fmt.Errorf("invalid events[%v].GroupID('%v') != events[0].GroupID('%v')", idx, event.GroupID(), groupID)
			}
			// timestamp values must be non-decreasing
			if timestamp.After(event.Timestamp()) {
				return fmt.Errorf("invalid decreasing events[%v].Timestamp(%v))", idx, event.Timestamp())
			}
			timestamp = event.Timestamp()
		}
	}
	return nil
}

// Save save events to eventstore.
// AggregateID, GroupID and EventType are required.
// All events within one Save operation shall have the same AggregateID and GroupID.
// Versions shall be unique and ascend continually.
// Only first event can be a snapshot.
func (s *EventStore) Save(ctx context.Context, events ...eventstore.Event) (eventstore.SaveStatus, error) {
	s.LogDebugfFunc("mongodb.Evenstore.Save start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.Save takes %v", time.Since(t))
	}()

	err := checkBeforeSave(events...)
	if err != nil {
		return eventstore.Fail, err
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())
	if events[0].Version() == 0 {
		doc, err := makeDBDoc(events, s.dataMarshaler)
		if err != nil {
			return eventstore.Fail, fmt.Errorf("cannot insert first events('%v'): %w", events, err)
		}
		_, err = col.InsertOne(ctx, doc)
		if err != nil {
			if IsDup(err) {
				return eventstore.ConcurrencyException, nil
			}
			return eventstore.Fail, fmt.Errorf("cannot insert first events('%v'): %w", events, err)
		}

		return eventstore.Ok, nil
	}
	status, err := s.saveEvent(ctx, col, events)
	if err != nil {
		return status, err
	}
	switch status {
	case eventstore.SnapshotRequired:
		if events[0].IsSnapshot() {
			return s.saveSnapshot(ctx, events)
		}
	}
	return status, nil
}

func (s *EventStore) saveSnapshot(ctx context.Context, events []eventstore.Event) (status eventstore.SaveStatus, err error) {
	doc, err := makeDBDoc(events, s.dataMarshaler)
	if err != nil {
		return eventstore.Fail, err
	}
	tx, err := s.client.StartSession()
	if err != nil {
		return eventstore.Fail, err
	}
	defer tx.EndSession(ctx)
	err = tx.StartTransaction()
	if err != nil {
		return eventstore.Fail, err
	}
	col := tx.Client().Database(s.DBName()).Collection(getEventCollectionName())
	opts := options.Update()
	opts.SetUpsert(true)
	opts.SetHint(aggregateIDLastVersionQueryIndex)

	// find document of aggregate with previous version
	// latestVersion shall be lower by 1 as new event otherwise other event was stored (occ).
	filter := bson.D{
		{Key: aggregateIDKey, Value: events[0].AggregateID()},
		{Key: latestVersionKey, Value: events[0].Version() - 1},
	}
	update := bson.M{
		"$set": bson.M{
			isActiveKey: false,
		},
	}
	res, err := col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if IsDup(err) {
			return eventstore.ConcurrencyException, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot remove current from doc for events('%v'): %w", events, err)
	}
	if res.ModifiedCount == 0 {
		return eventstore.ConcurrencyException, nil
	}

	_, err = col.InsertOne(ctx, doc)
	if err != nil {
		if IsDup(err) {
			return eventstore.ConcurrencyException, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot insert snapshot events('%v'): %w", events, err)
	}
	err = tx.CommitTransaction(ctx)
	if err != nil {
		if IsDup(err) {
			return eventstore.ConcurrencyException, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot commit transaction for events('%v'): %w", events, err)
	}
	return eventstore.Ok, nil
}

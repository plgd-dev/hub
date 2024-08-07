package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IsDup check it error is duplicate
func IsDup(err error) bool {
	// Besides being handy, helps with MongoDB bugs SERVER-7164 and SERVER-11493.
	// What follows makes me sad. Hopefully conventions will be more clear over time.
	var cErr mongo.CommandError
	if errors.As(err, &cErr) {
		return cErr.Code == 11000 || cErr.Code == 11001 || cErr.Code == 12582 || cErr.Code == 16460 && strings.Contains(cErr.Message, " E11000 ")
	}
	var wErr mongo.WriteError
	if errors.As(err, &wErr) {
		return wErr.Code == 11000 || wErr.Code == 11001 || wErr.Code == 12582
	}
	var wExp mongo.WriteException
	if errors.As(err, &wExp) {
		isDup := true
		for _, werr := range wExp.WriteErrors {
			if !IsDup(werr) {
				isDup = false
			}
		}
		return isDup
	}
	return false
}

func (s *EventStore) saveEvent(ctx context.Context, col *mongo.Collection, events []eventstore.Event) (status eventstore.SaveStatus, err error) {
	etag, types, e, err := makeDBEventsAndGetETag(events, s.dataMarshaler)
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
	if etag != nil {
		updateSet[latestETagKey] = makeDBETag(etag)
	}
	tryToSetServiceID(updateSet, events)
	if len(types) > 0 {
		updateSet[typesKey] = types
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
	switch {
	case err == nil:
		if res.ModifiedCount == 0 {
			return eventstore.ConcurrencyException, nil
		}
		return eventstore.Ok, nil
	case errors.Is(err, mongo.ErrNilDocument):
		return eventstore.ConcurrencyException, nil
	default:
		var wErr mongo.WriteException
		if errors.As(err, &wErr) {
			sizeIsExceeded := wErr.HasErrorCode(10334)
			if !sizeIsExceeded {
				return eventstore.Fail, fmt.Errorf("cannot push events('%v') to db: %w", events, err)
			}
			return eventstore.SnapshotRequired, nil
		}
		return eventstore.Fail, fmt.Errorf("cannot push events('%v') to db: %w", events, err)
	}
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

	if err := eventstore.ValidateEventsBeforeSave(events); err != nil {
		return eventstore.Fail, err
	}

	col := s.client().Database(s.DBName()).Collection(getEventCollectionName())
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
	if status == eventstore.SnapshotRequired && events[0].IsSnapshot() {
		return s.saveSnapshot(ctx, events)
	}
	return status, nil
}

func (s *EventStore) saveSnapshot(ctx context.Context, events []eventstore.Event) (status eventstore.SaveStatus, err error) {
	doc, err := makeDBDoc(events, s.dataMarshaler)
	if err != nil {
		return eventstore.Fail, err
	}
	tx, err := s.client().StartSession()
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

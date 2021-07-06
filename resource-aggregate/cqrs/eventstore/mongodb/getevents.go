package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/// Get array of unique aggregateId values
func getEventsAggregateIdFilter(queries []eventstore.GetEventsQuery) bson.A {
	if len(queries) == 0 {
		return nil
	}

	// get unique aggregateIds
	aggregateIds := make(map[string]struct{})
	for _, query := range queries {
		if len(query.AggregateID) != 0 {
			aggregateIds[query.AggregateID] = struct{}{}
		}
	}

	if len(aggregateIds) == 0 {
		return nil
	}

	// filter to include only given unique aggregateIds
	aggregateIdFilter := make(bson.A, 0, len(aggregateIds))
	for aggregateId := range aggregateIds {
		aggregateIdFilter = append(aggregateIdFilter, aggregateId)
	}
	return aggregateIdFilter
}

func getEventsFilter(groupID string, queries []eventstore.GetEventsQuery, timestamp int64) bson.D {
	filter := bson.D{}
	if len(groupID) > 0 {
		// filter documents by groupdID
		filter = append(filter, bson.E{Key: groupIDKey, Value: groupID})
	}

	if len(queries) == 0 {
		return filter
	}

	aggregateIdFilter := getEventsAggregateIdFilter(queries)
	if len(aggregateIdFilter) > 0 {
		// filter documents by aggregateID
		filter = append(filter, bson.E{
			Key: aggregateIDKey,
			Value: bson.M{
				"$in": aggregateIdFilter,
			},
		})
	}

	if timestamp > 0 {
		// filter documents that have the latest timestamp of events larger than given value
		filter = append(filter, bson.E{
			Key: latestTimestampKey,
			Value: bson.M{
				"$gt": timestamp,
			},
		})
	}

	return filter
}

func getEventsProjection(timestamp int64) bson.M {
	projection := bson.M{
		"_id":          0,
		groupIDKey:     1,
		aggregateIDKey: 1,
		eventsKey:      1,
	}

	if timestamp > 0 {
		filter := bson.M{
			"input": "$" + eventsKey,
			"as":    "event",
			"cond": bson.M{
				"$gt": bson.A{"$$event." + timestampKey, timestamp},
			},
		}
		projection[eventsKey] = bson.M{
			"$filter": filter,
		}
	}

	return projection
}

func getEventsQueriesToMongoQuery(groupID string, queries []eventstore.GetEventsQuery, timestamp int64) (interface{}, *options.FindOptions) {
	filter := getEventsFilter(groupID, queries, timestamp)

	opts := options.Find()
	opts.SetAllowDiskUse(true)
	opts.SetProjection(getEventsProjection(timestamp))

	return filter, opts
}

func (s *EventStore) getEvents(ctx context.Context, groupID string, queries []eventstore.GetEventsQuery, timestamp int64, eventHandler eventstore.Handler) error {
	filter, opts := getEventsQueriesToMongoQuery(groupID, queries, timestamp)
	return s.loadEventsQuery(ctx, eventHandler, nil, filter, opts)
}

func getNormalizedGetEventsQuery(queries []eventstore.GetEventsQuery) map[string][]eventstore.GetEventsQuery {
	normalizedQuery := make(map[string][]eventstore.GetEventsQuery)
	for _, query := range queries {
		v, ok := normalizedQuery[query.GroupID]
		if !ok {
			v = make([]eventstore.GetEventsQuery, 0, 1)
		}
		v = append(v, query)
		normalizedQuery[query.GroupID] = v
	}
	return normalizedQuery
}

// Get events from the eventstore.
func (s *EventStore) GetEvents(ctx context.Context, queries []eventstore.GetEventsQuery, timestamp int64, eventHandler eventstore.Handler) error {
	s.LogDebugfFunc("mongodb.Evenstore.GetEvents start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.GetEvents takes %v", time.Since(t))
	}()
	if len(queries) == 0 {
		return fmt.Errorf("not supported")
	}

	normalizedQuery := getNormalizedGetEventsQuery(queries)

	var errors []error
	for groupID, queries := range normalizedQuery {
		s.LogDebugfFunc("GroupID: %v, #queries: %v", groupID, len(queries))
		err := s.getEvents(ctx, groupID, queries, timestamp, eventHandler)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}

	return nil
}

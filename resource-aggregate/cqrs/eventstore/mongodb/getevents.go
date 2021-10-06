package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/strings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/// Get array of unique aggregateId values
func getEventsAggregateIdFilter(groupID string, queries []eventstore.GetEventsQuery) bson.A {
	if len(queries) == 0 {
		return nil
	}

	// get unique aggregateIds
	aggregateIds := make(map[string]struct{})
	for _, query := range queries {
		if query.GroupID == groupID && len(query.AggregateID) != 0 {
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

	if len(queries) > 0 {
		aggregateIdFilter := getEventsAggregateIdFilter(groupID, queries)
		if len(aggregateIdFilter) > 0 {
			// filter documents by aggregateID
			filter = append(filter, bson.E{
				Key: aggregateIDKey,
				Value: bson.M{
					"$in": aggregateIdFilter,
				},
			})
		}
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

func getEventsQueriesToMongoQuery(groupID string, queries []eventstore.GetEventsQuery, timestamp int64) (bson.D, *options.FindOptions) {
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

type ResourceIdFilter struct {
	All         bool
	ResourceIds strings.Set
}

type DeviceIdFilter struct {
	All       bool
	DeviceIds map[string]ResourceIdFilter
}

func GetNormalizedGetEventsFilter(queries []eventstore.GetEventsQuery) DeviceIdFilter {
	filter := DeviceIdFilter{
		All:       false,
		DeviceIds: make(map[string]ResourceIdFilter),
	}

	for _, query := range queries {
		if len(query.GroupID) == 0 && len(query.AggregateID) == 0 {
			return DeviceIdFilter{
				All: true,
			}
		}

		v, ok := filter.DeviceIds[query.GroupID]
		if !ok {
			v = ResourceIdFilter{
				All:         false,
				ResourceIds: make(strings.Set),
			}
		}
		if v.All {
			continue
		}

		if len(query.AggregateID) == 0 {
			v.All = true
			v.ResourceIds = nil
		} else {
			v.ResourceIds.Add(query.AggregateID)
		}
		filter.DeviceIds[query.GroupID] = v
	}
	return filter
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

	eventFilter := GetNormalizedGetEventsFilter(queries)
	if eventFilter.All {
		s.LogDebugfFunc("Query all events")
		return s.getEvents(ctx, "", nil, timestamp, eventHandler)
	}

	var errors []error
	for groupID, filter := range eventFilter.DeviceIds {
		s.LogDebugfFunc("GroupID: %v, all: %v #resourceIds: %v", groupID, filter.All, len(filter.ResourceIds))
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

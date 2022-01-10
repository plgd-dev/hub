package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/strings"
	"go.mongodb.org/mongo-driver/bson"
)

func getDeviceIdFilter(queries []eventstore.DeleteQuery) bson.A {
	deviceIds := make(strings.Set)
	for _, q := range queries {
		if q.GroupID != "" {
			deviceIds.Add(q.GroupID)
		}
	}

	deviceIdFilter := make(bson.A, 0, len(deviceIds))
	for deviceId := range deviceIds {
		deviceIdFilter = append(deviceIdFilter, deviceId)
	}

	return deviceIdFilter
}

// Delete documents with given group ids
func (s *EventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) error {
	deviceIdFilter := getDeviceIdFilter(queries)
	if len(deviceIdFilter) == 0 {
		return fmt.Errorf("failed to delete documents: invalid query")
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())

	_, err := col.DeleteMany(ctx, bson.M{
		groupIDKey: bson.M{
			"$in": deviceIdFilter,
		},
	})
	return err
}

package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/v2/strings"
	"go.mongodb.org/mongo-driver/bson"
)

func getDeviceIDFilter(queries []eventstore.DeleteQuery) bson.A {
	deviceIDs := make(strings.Set)
	for _, q := range queries {
		if q.GroupID != "" {
			deviceIDs.Add(q.GroupID)
		}
	}

	deviceIDFilter := make(bson.A, 0, len(deviceIDs))
	for deviceID := range deviceIDs {
		deviceIDFilter = append(deviceIDFilter, deviceID)
	}

	return deviceIDFilter
}

// Delete documents with given group ids
func (s *EventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) error {
	deviceIDFilter := getDeviceIDFilter(queries)
	if len(deviceIDFilter) == 0 {
		return fmt.Errorf("failed to delete documents: invalid query")
	}

	col := s.client().Database(s.DBName()).Collection(getEventCollectionName())

	_, err := col.DeleteMany(ctx, bson.M{
		groupIDKey: bson.M{
			"$in": deviceIDFilter,
		},
	})
	return err
}

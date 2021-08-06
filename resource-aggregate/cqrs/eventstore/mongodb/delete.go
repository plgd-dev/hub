package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
	"go.mongodb.org/mongo-driver/bson"
)

// Return set of unique non-empty device ids
func getUniqueDeviceIdsFromDeleteQuery(queries []eventstore.DeleteQuery) strings.Set {
	deviceIds := make(strings.Set)
	for _, q := range queries {
		if q.GroupID != "" {
			deviceIds.Add(q.GroupID)
		}
	}
	return deviceIds
}

// Delete documents with given group id and return slice of deleted ids.
// To be able to return the slice of deleted ids mongodb is queried in a loop and each iteration
// deletes only documents with group id of the iteration. If an error occurs then whole function
// is terminated and error is returned, however previous iterations are not reverted. Some
// documents might have been deleted, to get the current data you must query the database again.
func (s *EventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) ([]string, error) {
	deviceIds := getUniqueDeviceIdsFromDeleteQuery(queries)
	if len(deviceIds) == 0 {
		return nil, fmt.Errorf("failed to delete documents: invalid groupID")
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())

	var deletedDeviceIds []string
	for deviceId := range deviceIds {
		res, err := col.DeleteMany(ctx, bson.M{
			groupIDKey: deviceId,
		})
		if err != nil {
			return nil, err
		}
		if res.DeletedCount < 1 {
			continue
		}

		deletedDeviceIds = append(deletedDeviceIds, deviceId)
	}

	return deletedDeviceIds, nil
}

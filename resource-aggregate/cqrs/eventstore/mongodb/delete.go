package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
	"go.mongodb.org/mongo-driver/bson"
)

/// Return set of unique non-empty device ids
func getUniqueDeviceIds(queries []eventstore.DeleteQuery) strings.Set {
	deviceIds := make(strings.Set)
	for _, q := range queries {
		if q.GroupID != "" {
			deviceIds.Add(q.GroupID)
		}
	}
	return deviceIds
}

/// Delete documents with given group id and return slice of deleted ids.
/// To be able to return the slice of deleted ids mongodb is queried in a loop and each iteration
/// deletes only documents with group id of the iteration. If an error occurs then the current
/// iteration is skipped and error is saved, but the loop continues. Therefore, you must always
/// check both return values to known that all queries were executed without issues.
func (s *EventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) ([]string, error) {
	deviceIds := getUniqueDeviceIds(queries)
	if len(deviceIds) == 0 {
		return nil, fmt.Errorf("failed to delete documents: invalid groupid")
	}

	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())

	var deletedDeviceIds []string
	var errors []error
	for deviceId := range deviceIds {
		res, err := col.DeleteMany(ctx, bson.M{
			groupIDKey: deviceId,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to delete documents with groupid(%v): %v", deviceId, err))
			continue
		}
		if res.DeletedCount < 1 {
			continue
		}

		deletedDeviceIds = append(deletedDeviceIds, deviceId)
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("%+v", errors)
	}

	return deletedDeviceIds, err
}

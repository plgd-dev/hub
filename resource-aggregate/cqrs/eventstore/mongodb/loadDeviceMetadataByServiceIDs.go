package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/strings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DeviceDocumentMetadata struct {
	DeviceID  string
	ServiceID string
}

func (s *EventStore) LoadDeviceMetadataByServiceIDs(ctx context.Context, serviceIDs []string, limit int64) ([]DeviceDocumentMetadata, error) {
	s.LogDebugfFunc("mongodb.Evenstore.LoadDocMetadataFromByServiceIDs start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.LoadDocMetadataFromByServiceIDs takes %v", time.Since(t))
	}()
	if len(serviceIDs) == 0 {
		return nil, fmt.Errorf("not supported")
	}
	serviceIDs = strings.Unique(serviceIDs)

	opts := options.Find()
	opts.SetAllowDiskUse(true)
	opts.SetProjection(bson.M{
		groupIDKey:   1,
		serviceIDKey: 1,
	})
	opts.SetLimit(limit)
	opts.SetHint(serviceIDQueryIndex)

	filterService := make([]bson.D, 0, 32)
	for _, q := range serviceIDs {
		filterService = append(filterService, bson.D{{Key: serviceIDKey, Value: q}, {Key: isActiveKey, Value: true}})
	}
	filter := bson.M{
		"$or": filterService,
	}
	col := s.client.Database(s.DBName()).Collection(getEventCollectionName())
	iter, err := col.Find(ctx, filter, opts)
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ret := make([]DeviceDocumentMetadata, 0, iter.RemainingBatchLength())
	for iter.Next(ctx) {
		var doc bson.M
		err := iter.Decode(&doc)
		if err != nil {
			return nil, err
		}
		groupID, ok := doc[groupIDKey].(string)
		if !ok {
			return nil, fmt.Errorf(errFmtDataIsNotStringType, groupIDKey, doc[groupIDKey])
		}
		aggregateID, ok := doc[serviceIDKey].(string)
		if !ok {
			return nil, fmt.Errorf(errFmtDataIsNotStringType, aggregateIDKey, doc[serviceIDKey])
		}
		ret = append(ret, DeviceDocumentMetadata{
			DeviceID:  groupID,
			ServiceID: aggregateID,
		})
	}
	return ret, nil
}

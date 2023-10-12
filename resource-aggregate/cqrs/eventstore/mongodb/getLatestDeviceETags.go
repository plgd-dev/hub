package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func decodeETag(cur *mongo.Cursor) ([]byte, error) {
	var v map[string]interface{}
	err := cur.Decode(&v)
	if err != nil {
		return nil, err
	}
	latestETagRaw := v[latestETagKey]
	if latestETagRaw == nil {
		return nil, fmt.Errorf("cannot find '%v' in result %v", latestETagKey, v)
	}
	latestETag, ok := latestETagRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot convert latestETag %T to map[string]interface{}", latestETagRaw)
	}
	etagRaw := latestETag[etagKey]
	if etagRaw == nil {
		return nil, fmt.Errorf("cannot find '%v' in latestETagKey %v", etagKey, latestETag[etagKey])
	}
	etag, ok := etagRaw.(primitive.Binary)
	if !ok {
		return nil, fmt.Errorf("cannot convert etag %T to primitive.Binary", etagRaw)
	}
	if len(etag.Data) == 0 {
		return nil, fmt.Errorf("etag is empty")
	}
	return etag.Data, nil
}

// Get latest ETags for device resources from event store for batch observing
func (s *EventStore) GetLatestDeviceETags(ctx context.Context, deviceID string, limit uint32) ([][]byte, error) {
	s.LogDebugfFunc("mongodb.Evenstore.GetLatestETag start")
	t := time.Now()
	defer func() {
		s.LogDebugfFunc("mongodb.Evenstore.GetLatestETag takes %v", time.Since(t))
	}()
	if deviceID == "" {
		return nil, fmt.Errorf("deviceID is invalid")
	}
	filter := bson.D{
		bson.E{Key: groupIDKey, Value: deviceID},
	}
	col := s.client().Database(s.DBName()).Collection(getEventCollectionName())
	opts := options.Find().SetSort(bson.D{{Key: latestETagKeyTimestampKey, Value: -1}}).SetProjection(bson.D{{Key: latestETagKey, Value: 1}}).SetHint(groupIDETagLatestTimestampQueryIndex)
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cur, err := col.Find(ctx, filter, opts)
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	etags := make([][]byte, 0, limit)
	var errors *multierror.Error
	for cur.Next(ctx) {
		etag, err := decodeETag(cur)
		if err == nil {
			etags = append(etags, etag)
		} else {
			errors = multierror.Append(errors, err)
		}
	}
	if len(etags) > 0 {
		if errors.ErrorOrNil() != nil {
			s.LogDebugfFunc("cannot decode all resources etags: %v", errors.ErrorOrNil())
		}
		return etags, nil
	}
	return nil, errors.ErrorOrNil()
}

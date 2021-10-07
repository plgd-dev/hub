package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"

	pkgMongo "github.com/plgd-dev/hub/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	*pkgMongo.Store
}

func NewStore(ctx context.Context, cfg pkgMongo.Config, tls *tls.Config) (*Store, error) {
	s, err := pkgMongo.NewStoreWithCollection(ctx, cfg, tls, subscriptionsCName, typeQueryIndex, typeDeviceIDQueryIndex,
		typeResourceIDQueryIndex, typeInitializedIDQueryIndex)
	if err != nil {
		return nil, err
	}
	s.SetOnClear(func(c context.Context) error {
		return s.DropCollection(ctx, subscriptionsCName)
	})
	return &Store{s}, nil
}

func incrementSubscriptionSequenceNumber(ctx context.Context, col *mongo.Collection, subscriptionId string) (uint64, error) {
	if subscriptionId == "" {
		return 0, fmt.Errorf("cannot increment sequence number: invalid subscriptionId")
	}

	var res bson.M
	opts := &options.FindOneAndUpdateOptions{}
	result := col.FindOneAndUpdate(ctx, bson.M{"_id": subscriptionId}, bson.M{"$inc": bson.M{sequenceNumberKey: 1}}, opts.SetReturnDocument(options.After))
	if result.Err() != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionId, result.Err())
	}

	err := result.Decode(&res)
	if err != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionId, err)
	}

	value, ok := res[sequenceNumberKey]
	if !ok {
		return 0, fmt.Errorf("cannot increment sequence number for %v: '%v' not found", subscriptionId, sequenceNumberKey)
	}

	return uint64(value.(int64)) - 1, nil
}

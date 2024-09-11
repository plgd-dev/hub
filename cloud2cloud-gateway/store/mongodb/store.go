package mongodb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/internal/math"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
}

func NewStore(ctx context.Context, cfg pkgMongo.Config, tls *tls.Config, tracerProvider trace.TracerProvider) (*Store, error) {
	s, err := pkgMongo.NewStoreWithCollection(ctx, &cfg, tls, tracerProvider, subscriptionsCName, typeQueryIndex, typeDeviceIDQueryIndex,
		typeResourceIDQueryIndex, typeInitializedIDQueryIndex)
	if err != nil {
		return nil, err
	}
	s.SetOnClear(func(clearCtx context.Context) error {
		return s.DropCollection(clearCtx, subscriptionsCName)
	})
	return &Store{s}, nil
}

func incrementSubscriptionSequenceNumber(ctx context.Context, col *mongo.Collection, subscriptionID string) (uint64, error) {
	if subscriptionID == "" {
		return 0, errors.New("cannot increment sequence number: invalid subscriptionID")
	}

	var res bson.M
	opts := &options.FindOneAndUpdateOptions{}
	result := col.FindOneAndUpdate(ctx, bson.M{"_id": subscriptionID}, bson.M{"$inc": bson.M{sequenceNumberKey: 1}}, opts.SetReturnDocument(options.After))
	if result.Err() != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionID, result.Err())
	}

	err := result.Decode(&res)
	if err != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionID, err)
	}

	value, ok := res[sequenceNumberKey]
	if !ok {
		return 0, fmt.Errorf("cannot increment sequence number for %v: '%v' not found", subscriptionID, sequenceNumberKey)
	}

	return math.CastTo[uint64](value.(int64)) - 1, nil
}

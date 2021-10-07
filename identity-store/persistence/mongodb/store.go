package mongodb

import (
	"context"
	"crypto/tls"

	pkgMongo "github.com/plgd-dev/hub/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
)

const userDevicesCName = "userdevices"

var userDeviceQueryIndex = bson.D{
	{Key: ownerKey, Value: 1},
	{Key: deviceIDKey, Value: 1},
}

var userDevicesQueryIndex = bson.D{
	{Key: ownerKey, Value: 1},
}

type Store struct {
	*pkgMongo.Store
}

func NewStore(ctx context.Context, cfg pkgMongo.Config, tls *tls.Config) (*Store, error) {
	s, err := pkgMongo.NewStoreWithCollection(ctx, cfg, tls, userDevicesCName, userDeviceQueryIndex, userDevicesQueryIndex)
	if err != nil {
		return nil, err
	}
	return &Store{s}, nil
}

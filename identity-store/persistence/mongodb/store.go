package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

const userDevicesCName = "userdevices"

var userDeviceQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: ownerKey, Value: 1},
		{Key: deviceIDKey, Value: 1},
	},
}

var userDevicesQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: ownerKey, Value: 1},
	},
}

type Store struct {
	*pkgMongo.Store
}

func New(ctx context.Context, config *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(config.TLS, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}
	s, err := pkgMongo.NewStoreWithCollection(ctx, config, certManager.GetTLSConfig(), tracerProvider, userDevicesCName, userDeviceQueryIndex, userDevicesQueryIndex)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s.AddCloseFunc(certManager.Close)
	return &Store{s}, nil
}

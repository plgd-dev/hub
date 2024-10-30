package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
}

const (
	conditionsCol            = "conditions"
	configurationsCol        = "configurations"
	appliedConfigurationsCol = "appliedConfigurations"
)

var idUniqueIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: pb.IDKey, Value: 1},
	},
	Options: options.Index().SetUnique(true),
}

var deviceIDConfigurationIDUniqueIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: pb.DeviceIDKey, Value: 1},
		{Key: pb.ConfigurationLinkIDKey, Value: 1},
	},
	Options: options.Index().SetUnique(true),
}

func New(ctx context.Context, cfg *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(cfg.Mongo.TLS, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}

	m, err := pkgMongo.NewStoreWithCollections(ctx, &cfg.Mongo, certManager.GetTLSConfig(), tracerProvider, map[string][]mongo.IndexModel{
		conditionsCol:            nil,
		configurationsCol:        nil,
		appliedConfigurationsCol: {idUniqueIndex, deviceIDConfigurationIDUniqueIndex},
	})
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s := Store{Store: m}
	s.SetOnClear(s.clearDatabases)
	s.AddCloseFunc(certManager.Close)
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	var errors *multierror.Error
	collections := []string{conditionsCol, configurationsCol, appliedConfigurationsCol}
	for _, collection := range collections {
		err := s.Collection(collection).Drop(ctx)
		errors = multierror.Append(errors, err)
	}
	return errors.ErrorOrNil()
}

func (s *Store) Close(ctx context.Context) error {
	return s.Store.Close(ctx)
}

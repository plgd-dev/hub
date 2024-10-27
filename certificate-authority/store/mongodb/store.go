package mongodb

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
	logger log.Logger
}

var deviceIDKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.DeviceIDKey, Value: 1},
		{Key: store.OwnerKey, Value: 1},
	},
}

var commonNameKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.CommonNameKey, Value: 1},
		{Key: store.OwnerKey, Value: 1},
	},
}

type MongoIterator[T any] struct {
	Cursor *mongo.Cursor
}

func (i *MongoIterator[T]) Next(ctx context.Context, s *T) bool {
	if !i.Cursor.Next(ctx) {
		return false
	}
	err := i.Cursor.Decode(s)
	return err == nil
}

func (i *MongoIterator[T]) Err() error {
	return i.Cursor.Err()
}

func processCursor[T any](ctx context.Context, cr *mongo.Cursor, p store.Process[T]) (int, error) {
	var errors *multierror.Error
	iter := MongoIterator[T]{
		Cursor: cr,
	}
	count := 0
	for {
		var stored T
		if !iter.Next(ctx, &stored) {
			break
		}
		err := p(&stored)
		if err != nil {
			errors = multierror.Append(errors, err)
			break
		}
		count++
	}
	errors = multierror.Append(errors, iter.Err())
	errClose := cr.Close(ctx)
	errors = multierror.Append(errors, errClose)
	return count, errors.ErrorOrNil()
}

func New(ctx context.Context, cfg *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(cfg.Mongo.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}
	m, err := pkgMongo.NewStoreWithCollections(ctx, &cfg.Mongo, certManager.GetTLSConfig(), tracerProvider, map[string][]mongo.IndexModel{
		signingRecordsCol: {commonNameKeyQueryIndex, deviceIDKeyQueryIndex},
		revocationListCol: nil,
	})
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s := Store{
		Store:  m,
		logger: logger,
	}
	s.SetOnClear(s.clearDatabases)
	s.AddCloseFunc(certManager.Close)
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	var errs *multierror.Error
	errs = multierror.Append(errs, s.Collection(signingRecordsCol).Drop(ctx))
	errs = multierror.Append(errs, s.Collection(revocationListCol).Drop(ctx))
	return errs.ErrorOrNil()
}

func (s *Store) Close(ctx context.Context) error {
	return s.Store.Close(ctx)
}

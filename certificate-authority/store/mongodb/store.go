package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
	bulkWriter *bulkWriter
}

var DeviceIDKeyQueryIndex = bson.D{
	{Key: store.DeviceIDKey, Value: 1},
	{Key: store.OwnerKey, Value: 1},
}

var CommonNameKeyQueryIndex = bson.D{
	{Key: store.CommonNameKey, Value: 1},
	{Key: store.OwnerKey, Value: 1},
}

var PublicKeyQueryIndex = bson.D{
	{Key: store.CommonNameKey, Value: 1},
}

func New(ctx context.Context, cfg *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(cfg.Mongo.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}
	m, err := pkgMongo.NewStore(ctx, &cfg.Mongo, certManager.GetTLSConfig(), tracerProvider)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	bulkWriter := newBulkWriter(m.Collection(signingRecordsCol), cfg.BulkWrite.DocumentLimit, cfg.BulkWrite.ThrottleTime, cfg.BulkWrite.Timeout, logger)
	s := Store{Store: m, bulkWriter: bulkWriter}
	err = s.EnsureIndex(ctx, signingRecordsCol, CommonNameKeyQueryIndex, DeviceIDKeyQueryIndex)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s.SetOnClear(s.clearDatabases)
	s.AddCloseFunc(certManager.Close)
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	return s.Collection(signingRecordsCol).Drop(ctx)
}

func (s *Store) Close(ctx context.Context) error {
	s.bulkWriter.Close()
	return s.Store.Close(ctx)
}

func (s *Store) FlushBulkWriter() error {
	_, err := s.bulkWriter.bulkWrite()
	return err
}

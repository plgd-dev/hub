package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
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

func NewStore(ctx context.Context, cfg Config, tls *tls.Config, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	m, err := pkgMongo.NewStore(ctx, cfg.Mongo, tls, tracerProvider)
	if err != nil {
		return nil, err
	}
	bulkWriter := newBulkWriter(m.Collection(signingRecordsCol), cfg.BulkWrite.DocumentLimit, cfg.BulkWrite.ThrottleTime, cfg.BulkWrite.Timeout, logger)
	s := Store{Store: m, bulkWriter: bulkWriter}
	err = s.EnsureIndex(ctx, signingRecordsCol, CommonNameKeyQueryIndex, DeviceIDKeyQueryIndex)
	if err != nil {
		return nil, err
	}
	s.SetOnClear(func(c context.Context) error {
		return s.clearDatabases(ctx)
	})
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	var errors []error
	if err := s.Collection(signingRecordsCol).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot clear: %v", errors)
	}
	return nil
}

func (s *Store) Close(ctx context.Context) error {
	s.bulkWriter.Close()
	return s.Store.Close(ctx)
}

func (s *Store) FlushBulkWriter() error {
	_, err := s.bulkWriter.bulkWrite()
	return err
}

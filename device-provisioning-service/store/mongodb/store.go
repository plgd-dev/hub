package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
	bulkWriter *bulkWriter
}

var EnrollmentGroupIDKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.EnrollmentGroupIDKey, Value: 1},
	},
}

var DeviceIDKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.DeviceIDKey, Value: 1},
	},
}

var AttestationMechanismX509CertificateNamesKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.AttestationMechanismX509LeadCertificateNameKey, Value: 1},
	},
}

var AttestationMechanismX509CertificateNamesOwnerKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.AttestationMechanismX509LeadCertificateNameKey, Value: 1},
		{Key: store.OwnerKey, Value: 1},
	},
}

var IDKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.IDKey, Value: 1},
	},
}

var IDOwnerKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.IDKey, Value: 1},
		{Key: store.OwnerKey, Value: 1},
	},
}

var HubIDKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.HubIDsKey, Value: 1},
	},
}

var HubIDOwnerKeyQueryIndex = mongo.IndexModel{
	Keys: bson.D{
		{Key: store.HubIDsKey, Value: 1},
		{Key: store.OwnerKey, Value: 1},
	},
}

func NewStore(ctx context.Context, cfg Config, tls *tls.Config, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	m, err := pkgMongo.NewStore(ctx, &cfg.Mongo, tls, tracerProvider)
	if err != nil {
		return nil, err
	}
	bulkWriter := newBulkWriter(m.Collection(provisionedRecordsCol), cfg.BulkWrite.DocumentLimit, cfg.BulkWrite.ThrottleTime, cfg.BulkWrite.Timeout, logger)
	s := Store{Store: m, bulkWriter: bulkWriter}
	err = s.EnsureIndex(ctx, provisionedRecordsCol, EnrollmentGroupIDKeyQueryIndex, DeviceIDKeyQueryIndex)
	if err != nil {
		return nil, err
	}
	err = s.EnsureIndex(ctx, enrollmentGroupsCol, AttestationMechanismX509CertificateNamesOwnerKeyQueryIndex, AttestationMechanismX509CertificateNamesKeyQueryIndex, HubIDOwnerKeyQueryIndex, IDOwnerKeyQueryIndex, HubIDKeyQueryIndex)
	if err != nil {
		return nil, err
	}

	s.SetOnClear(s.clearDatabases)
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	var errors []error
	if err := s.Collection(provisionedRecordsCol).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if err := s.Collection(enrollmentGroupsCol).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if err := s.Collection(hubsCol).Drop(ctx); err != nil {
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

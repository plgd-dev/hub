package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const provisionedRecordsCol = "provisionedRecords"

func validateProvisioningRecord(owner string, provisionedRecord *store.ProvisioningRecord) error {
	if provisionedRecord.GetId() == "" {
		return errors.New("empty provisioned record ID")
	}
	if owner != "" && provisionedRecord.GetOwner() != owner {
		return fmt.Errorf("owner('%v') expects %v", provisionedRecord.GetOwner(), owner)
	}
	if provisionedRecord.GetAttestation() != nil && provisionedRecord.GetAttestation().GetDate() == 0 {
		return errors.New("empty attestation date")
	}
	if provisionedRecord.GetAcl() != nil && provisionedRecord.GetAcl().GetStatus().GetDate() == 0 {
		return errors.New("empty ACL date")
	}
	if provisionedRecord.GetCloud() != nil && provisionedRecord.GetCloud().GetStatus().GetDate() == 0 {
		return errors.New("empty cloud date")
	}
	if provisionedRecord.GetCredential() != nil && provisionedRecord.GetCredential().GetStatus().GetDate() == 0 {
		return errors.New("empty credential status date")
	}
	return nil
}

func (s *Store) UpdateProvisioningRecord(_ context.Context, owner string, provisionedRecord *store.ProvisioningRecord) error {
	if err := validateProvisioningRecord(owner, provisionedRecord); err != nil {
		return err
	}
	s.bulkWriter.push(provisionedRecord)
	return nil
}

func toProvisioningRecordsQueryFilter(owner string, queries *store.ProvisioningRecordsQuery) bson.D {
	or := []bson.D{}
	for _, q := range queries.GetIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{{Key: store.IDKey, Value: q}}))
	}
	for _, q := range queries.GetEnrollmentGroupIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{{Key: store.EnrollmentGroupIDKey, Value: q}}))
	}
	for _, q := range queries.GetDeviceIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{{Key: store.DeviceIDKey, Value: q}}))
	}
	switch len(or) {
	case 0:
		return addOwnerToFilter(owner, bson.D{})
	case 1:
		return or[0]
	}
	return bson.D{{Key: "$or", Value: or}}
}

func (s *Store) DeleteProvisioningRecords(ctx context.Context, owner string, query *store.ProvisioningRecordsQuery) (int64, error) {
	q := toProvisioningRecordsQueryFilter(owner, query)
	res, err := s.Collection(provisionedRecordsCol).DeleteMany(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("cannot remove device provision records for owner %v: %w", owner, err)
	}
	return res.DeletedCount, nil
}

func (s *Store) LoadProvisioningRecords(ctx context.Context, owner string, query *store.ProvisioningRecordsQuery, h store.LoadProvisioningRecordsFunc) error {
	col := s.Collection(provisionedRecordsCol)
	iter, err := col.Find(ctx, toProvisioningRecordsQueryFilter(owner, query))
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil
	}
	if err != nil {
		return err
	}

	i := provisioningRecordsIterator{
		iter: iter,
	}
	err = h(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type provisioningRecordsIterator struct {
	iter *mongo.Cursor
	err  error
}

func (i *provisioningRecordsIterator) Next(ctx context.Context, s *store.ProvisioningRecord) bool {
	if !i.iter.Next(ctx) {
		return false
	}
	err := i.iter.Decode(s)
	if err != nil {
		i.err = err
		return false
	}
	return true
}

func (i *provisioningRecordsIterator) Err() error {
	if i.err != nil {
		return i.err
	}
	return i.iter.Err()
}

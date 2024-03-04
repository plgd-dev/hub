package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const signingRecordsCol = "signedCertificateRecords"

var ErrCannotRemoveSigningRecord = errors.New("cannot remove signing record")

func (s *Store) CreateSigningRecord(ctx context.Context, signingRecord *store.SigningRecord) error {
	if err := signingRecord.Validate(); err != nil {
		return err
	}
	_, err := s.Collection(signingRecordsCol).InsertOne(ctx, signingRecord)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateSigningRecord(_ context.Context, signingRecord *store.SigningRecord) error {
	if err := signingRecord.Validate(); err != nil {
		return err
	}

	s.bulkWriter.Push(signingRecord)
	return nil
}

func toCommonNameQueryFilter(owner string, commonName string) bson.D {
	f := bson.D{
		{Key: store.CommonNameKey, Value: commonName},
	}
	if owner != "" {
		f = append(f, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return f
}

func toDeviceIDQueryFilter(owner string, deviceID string) bson.D {
	f := bson.D{
		{Key: store.DeviceIDKey, Value: deviceID},
	}
	if owner != "" {
		f = append(f, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return f
}

func toIDQueryFilter(owner string, id string) bson.D {
	f := bson.D{
		{Key: "_id", Value: id},
	}
	if owner != "" {
		f = append(f, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return f
}

func toSigningRecordsQueryFilter(owner string, queries *store.SigningRecordsQuery) interface{} {
	or := []bson.D{}
	for _, q := range queries.GetIdFilter() {
		or = append(or, toIDQueryFilter(owner, q))
	}
	for _, q := range queries.GetCommonNameFilter() {
		or = append(or, toCommonNameQueryFilter(owner, q))
	}
	for _, q := range queries.GetDeviceIdFilter() {
		or = append(or, toDeviceIDQueryFilter(owner, q))
	}
	switch len(or) {
	case 0:
		return bson.D{}
	case 1:
		return or[0]
	}
	return bson.M{"$or": or}
}

func (s *Store) DeleteSigningRecords(ctx context.Context, owner string, query *store.DeleteSigningRecordsQuery) (int64, error) {
	q := store.SigningRecordsQuery{
		IdFilter:       query.GetIdFilter(),
		DeviceIdFilter: query.GetDeviceIdFilter(),
	}
	res, err := s.Collection(signingRecordsCol).DeleteMany(ctx, toSigningRecordsQueryFilter(owner, &q))
	if err != nil {
		return -1, multierror.Append(ErrCannotRemoveSigningRecord, err)
	}
	return res.DeletedCount, nil
}

func (s *Store) DeleteNonDeviceExpiredRecords(ctx context.Context, now time.Time) (int64, error) {
	t := now.UnixNano()
	res, err := s.Collection(signingRecordsCol).DeleteMany(ctx, bson.M{
		store.CredentialKey + "." + store.ValidUntilDateKey: bson.M{"$lt": t},
		store.DeviceIDKey: bson.M{"$exists": false},
	})
	if err != nil {
		return -1, multierror.Append(ErrCannotRemoveSigningRecord, err)
	}

	return res.DeletedCount, nil
}

func (s *Store) LoadSigningRecords(ctx context.Context, owner string, query *store.SigningRecordsQuery, h store.LoadSigningRecordsFunc) error {
	col := s.Collection(signingRecordsCol)
	iter, err := col.Find(ctx, toSigningRecordsQueryFilter(owner, query))
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil
	}
	if err != nil {
		return err
	}

	i := SigningRecordsIterator{
		iter: iter,
	}
	err = h(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type SigningRecordsIterator struct {
	iter *mongo.Cursor
}

func (i *SigningRecordsIterator) Next(ctx context.Context, s *store.SigningRecord) bool {
	if !i.iter.Next(ctx) {
		return false
	}
	err := i.iter.Decode(s)
	return err == nil
}

func (i *SigningRecordsIterator) Err() error {
	return i.iter.Err()
}

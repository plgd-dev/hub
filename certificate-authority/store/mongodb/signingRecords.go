package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/maps"
)

const signingRecordsCol = "signedCertificateRecords"

var ErrCannotRemoveSigningRecord = errors.New("cannot remove signing record")

func (s *Store) CreateSigningRecord(ctx context.Context, signingRecord *store.SigningRecord) error {
	if err := signingRecord.Validate(); err != nil {
		return err
	}
	_, err := s.Collection(signingRecordsCol).InsertOne(ctx, signingRecord)
	return err
}

func (s *Store) UpdateSigningRecord(ctx context.Context, signingRecord *store.SigningRecord) error {
	if err := signingRecord.Validate(); err != nil {
		return err
	}
	filter := bson.M{"_id": signingRecord.GetId()}
	upsert := true
	opts := &options.UpdateOptions{
		Upsert: &upsert,
	}
	_, err := s.Collection(signingRecordsCol).UpdateOne(ctx, filter, bson.M{"$set": signingRecord}, opts)
	return err
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
		if owner == "" {
			return bson.D{}
		}
		return bson.D{
			{Key: store.OwnerKey, Value: owner},
		}
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

func (s *Store) RevokeSigningRecords(ctx context.Context, ownerID string, query *store.RevokeSigningRecordsQuery) (int64, error) {
	now := time.Now().UnixNano()
	// get signing records to be deleted
	type issuersRecord struct {
		ids          []string
		certificates []*store.RevocationListCertificate
	}
	idFilter := make(map[string]struct{})
	irs := make(map[string]issuersRecord)
	err := s.LoadSigningRecords(ctx, ownerID, &pb.GetSigningRecordsRequest{
		IdFilter:       query.GetIdFilter(),
		DeviceIdFilter: query.GetDeviceIdFilter(),
	}, func(v *pb.SigningRecord) error {
		credential := v.GetCredential()
		if credential == nil {
			return nil
		}
		idFilter[v.GetId()] = struct{}{}
		if credential.GetValidUntilDate() <= now {
			return nil
		}
		record := irs[credential.GetIssuerId()]
		record.ids = append(record.ids, v.GetId())
		record.certificates = append(record.certificates, &store.RevocationListCertificate{
			Serial:     credential.GetSerial(),
			ValidUntil: credential.GetValidUntilDate(),
			Revocation: now,
		})
		irs[credential.GetIssuerId()] = record
		return nil
	})
	if err != nil {
		return 0, err
	}

	// add certificates for the signing records to revocation lists
	for issuerID, record := range irs {
		if issuerID == "" {
			// no issuer id - for old records
			continue
		}
		query := store.UpdateRevocationListQuery{
			IssuerID:            issuerID,
			RevokedCertificates: record.certificates,
		}
		_, err := s.UpdateRevocationList(ctx, &query)
		if err != nil {
			return 0, err
		}
	}

	if len(idFilter) == 0 {
		return 0, nil
	}

	// delete the signing records
	return s.DeleteSigningRecords(ctx, ownerID, &pb.DeleteSigningRecordsRequest{
		IdFilter: maps.Keys(idFilter),
	})
}

func (s *Store) LoadSigningRecords(ctx context.Context, owner string, query *store.SigningRecordsQuery, p store.Process[store.SigningRecord]) error {
	col := s.Collection(signingRecordsCol)
	cur, err := col.Find(ctx, toSigningRecordsQueryFilter(owner, query))
	if err != nil {
		if errors.Is(err, mongo.ErrNilDocument) {
			return nil
		}
		return err
	}
	_, err = processCursor(ctx, cur, p)
	return err
}

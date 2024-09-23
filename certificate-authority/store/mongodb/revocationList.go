package mongodb

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/maps"
)

const revocationListCol = "revocationList"

func (s *Store) SupportsRevocationList() bool {
	return true
}

func (s *Store) InsertRevocationLists(ctx context.Context, rls ...*store.RevocationList) error {
	documents := make([]interface{}, 0, len(rls))
	for _, rl := range rls {
		if err := rl.Validate(); err != nil {
			return err
		}
		documents = append(documents, rl)
	}
	_, err := s.Collection(revocationListCol).InsertMany(ctx, documents)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("%w: %w", store.ErrDuplicateID, err)
	}
	return err
}

type revocationListUpdate struct {
	originalRevocationList *store.RevocationList
	certificatesToInsert   map[string]*store.RevocationListCertificate
}

// check the database and remove serials that are already in the array
func (s *Store) getRevocationListUpdate(ctx context.Context, query *store.UpdateRevocationListQuery) (revocationListUpdate, bool, error) {
	cmap := make(map[string]*store.RevocationListCertificate)
	for _, cert := range query.RevokedCertificates {
		if _, ok := cmap[cert.Serial]; ok {
			s.logger.Debugf("skipping duplicate serial number(%v) in query", cert.Serial)
			continue
		}
		if err := cert.Validate(); err != nil {
			return revocationListUpdate{}, false, err
		}
		cmap[cert.Serial] = cert
	}
	pl := mongo.Pipeline{
		bson.D{{Key: mongodb.Match, Value: bson.D{{Key: "_id", Value: query.IssuerID}}}},
	}
	if len(cmap) > 0 {
		pl = append(pl, bson.D{{Key: "$addFields", Value: bson.M{
			"duplicates": bson.M{
				"$filter": bson.M{
					"input": "$" + store.CertificatesKey,
					"as":    "cert",
					"cond":  bson.M{mongodb.In: bson.A{"$$cert." + store.SerialKey, maps.Keys(cmap)}},
				},
			},
		}}})
	}
	cur, err := s.Collection(revocationListCol).Aggregate(ctx, pl)
	if err != nil {
		return revocationListUpdate{}, false, err
	}
	type revocationListWithNewCertificates struct {
		*store.RevocationList `bson:",inline"`
		Duplicates            []*store.RevocationListCertificate `bson:"duplicates,omitempty"`
	}
	var rl *revocationListWithNewCertificates
	count, err := processCursor(ctx, cur, func(item *revocationListWithNewCertificates) error {
		rl = item
		return nil
	})
	if err != nil {
		return revocationListUpdate{}, false, err
	}
	if count == 0 {
		return revocationListUpdate{
			certificatesToInsert: cmap,
		}, true, nil
	}
	for _, c := range rl.Duplicates {
		s.logger.Debugf("skipping duplicate serial number(%v)", c.Serial)
		delete(cmap, c.Serial)
	}
	if len(cmap) == 0 && (!query.UpdateIfExpired || !rl.IsExpired()) {
		return revocationListUpdate{
			originalRevocationList: rl.RevocationList,
		}, false, nil
	}
	return revocationListUpdate{
		originalRevocationList: rl.RevocationList,
		certificatesToInsert:   cmap,
	}, true, nil
}

func (s *Store) UpdateRevocationList(ctx context.Context, query *store.UpdateRevocationListQuery) (*store.RevocationList, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	upd, needsUpdate, err := s.getRevocationListUpdate(ctx, query)
	if err != nil {
		return nil, err
	}
	if !needsUpdate {
		return upd.originalRevocationList, nil
	}

	if upd.originalRevocationList == nil {
		newRL := &store.RevocationList{
			Id:           query.IssuerID,
			Number:       "1", // the sequence for the CRL number field starts from 1
			IssuedAt:     query.IssuedAt,
			ValidUntil:   query.ValidUntil,
			Certificates: maps.Values(upd.certificatesToInsert),
		}
		if err = s.InsertRevocationLists(ctx, newRL); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, fmt.Errorf("%w: %w", store.ErrDuplicateID, err)
			}
			return nil, err
		}
		return newRL, nil
	}

	number, err := store.ParseBigInt(upd.originalRevocationList.Number)
	if err != nil {
		return nil, err
	}
	filter := bson.M{
		"_id":           query.IssuerID,
		store.NumberKey: number.String(),
	}

	nextNumber := number
	// for not issued (IssuedAt == 0) we don't need to increment the Number, it was already incremented when
	// the list was updated and the IssuedAt was set to 0
	if upd.originalRevocationList.IssuedAt != 0 {
		nextNumber = nextNumber.Add(nextNumber, big.NewInt(1))
	}
	update := bson.M{
		"$set": bson.M{
			store.NumberKey:     nextNumber.String(),
			store.IssuedAtKey:   query.IssuedAt,
			store.ValidUntilKey: query.ValidUntil,
		},
	}
	if len(upd.certificatesToInsert) > 0 {
		update["$push"] = bson.M{
			store.CertificatesKey: bson.M{"$each": maps.Values(upd.certificatesToInsert)},
		}
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedRL store.RevocationList
	if err = s.Collection(revocationListCol).FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedRL); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("%w: %w", store.ErrNotFound, err)
		}
		return nil, err
	}
	return &updatedRL, nil
}

func (s *Store) GetRevocationList(ctx context.Context, issuerID string, includeExpired bool) (*store.RevocationList, error) {
	now := time.Now().UnixNano()
	filter := bson.M{
		"_id": issuerID,
	}
	var opts []*options.FindOneOptions
	if !includeExpired {
		filter[store.CertificatesKey] = bson.M{
			"$elemMatch": bson.M{
				store.ValidUntilKey: bson.M{"$gte": now}, // non-expired certificates
			},
		}
		projection := bson.M{
			"_id":               1,
			store.NumberKey:     1,
			store.IssuedAtKey:   1,
			store.ValidUntilKey: 1,
			store.CertificatesKey: bson.M{
				"$filter": bson.M{
					"input": "$" + store.CertificatesKey,
					"as":    "cert",
					"cond": bson.M{
						"$gte": []interface{}{"$$cert." + store.ValidUntilKey, now}, // non-expired certificates
					},
				},
			},
		}
		opts = append(opts, options.FindOne().SetProjection(projection))
	}

	var rl store.RevocationList
	err := s.Collection(revocationListCol).FindOne(ctx, filter, opts...).Decode(&rl)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &rl, nil
}

func (s *Store) GetLatestIssuedOrIssueRevocationList(ctx context.Context, issuerID string, validFor time.Duration) (*store.RevocationList, error) {
	rl, err := s.GetRevocationList(ctx, issuerID, true)
	if err != nil {
		return nil, err
	}
	if rl.IssuedAt > 0 && !rl.IsExpired() {
		return rl, nil
	}
	issuedAt := time.Now()
	validUntil := issuedAt.Add(validFor)
	return s.UpdateRevocationList(ctx, &store.UpdateRevocationListQuery{
		IssuerID:        issuerID,
		IssuedAt:        issuedAt.UnixNano(),
		ValidUntil:      validUntil.UnixNano(),
		UpdateIfExpired: true,
	})
}

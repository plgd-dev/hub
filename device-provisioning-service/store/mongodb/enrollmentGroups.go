package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const enrollmentGroupsCol = "enrollmentGroups"

func (s *Store) CreateEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *store.EnrollmentGroup) error {
	if err := enrollmentGroup.Validate(owner); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	_, err := s.Collection(enrollmentGroupsCol).InsertOne(ctx, enrollmentGroup)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpsertEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *store.EnrollmentGroup) error {
	return s.updateEnrollmentGroup(ctx, owner, enrollmentGroup, true)
}

func (s *Store) UpdateEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *store.EnrollmentGroup) error {
	return s.updateEnrollmentGroup(ctx, owner, enrollmentGroup, false)
}

func (s *Store) updateEnrollmentGroup(ctx context.Context, owner string, enrollmentGroup *store.EnrollmentGroup, upsert bool) error {
	if err := enrollmentGroup.Validate(owner); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	filter, _ := toEnrollmentGroupFilter(owner, &store.EnrollmentGroupsQuery{
		IdFilter: []string{enrollmentGroup.GetId()},
	})
	res, err := s.Collection(enrollmentGroupsCol).UpdateOne(ctx, filter, bson.M{"$set": enrollmentGroup}, options.Update().SetUpsert(upsert))
	if err != nil {
		return err
	}
	if res.UpsertedCount > 0 && upsert {
		return nil
	}
	if res.ModifiedCount == 0 && res.MatchedCount == 0 {
		return mongo.ErrNilDocument
	}
	return nil
}

func addOwnerToFilter(owner string, q bson.D) bson.D {
	if owner != "" {
		return append(q, bson.E{Key: store.OwnerKey, Value: owner})
	}
	return q
}

func toEnrollmentGroupIDFilter(owner string, queries *store.EnrollmentGroupsQuery) ([]bson.D, *mongo.IndexModel) {
	or := []bson.D{}
	for _, q := range queries.GetIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{
			{Key: store.IDKey, Value: q},
		}))
	}
	var hint *mongo.IndexModel
	if len(or) > 0 {
		if owner != "" {
			hint = &IDOwnerKeyQueryIndex
		} else {
			hint = &IDKeyQueryIndex
		}
	}
	return or, hint
}

func toEnrollmentGroupHubIDFilter(owner string, queries *store.EnrollmentGroupsQuery, disableHintIn bool, or []bson.D, hintIn *mongo.IndexModel) (filter []bson.D, hint *mongo.IndexModel, disableHint bool) {
	disableHint = disableHintIn
	hint = hintIn
	if len(queries.GetHubIdFilter()) == 0 {
		return or, hint, disableHint
	}
	or = append(or, addOwnerToFilter(owner, bson.D{
		{Key: store.HubIDsKey, Value: bson.M{
			"$in": queries.GetHubIdFilter(),
		}},
	}))
	if hint == nil {
		if owner != "" {
			hint = &HubIDOwnerKeyQueryIndex
		} else {
			hint = &HubIDKeyQueryIndex
		}
	} else {
		hint = nil
		disableHint = true
	}
	return or, hint, disableHint
}

func toEnrollmentGroupAttestationMechanismX509CertificateNamesFilter(owner string, queries *store.EnrollmentGroupsQuery, disableHintIn bool, or []bson.D, hintIn *mongo.IndexModel) (filter []bson.D, hint *mongo.IndexModel, disableHint bool) {
	disableHint = disableHintIn
	hint = hintIn
	if len(queries.GetAttestationMechanismX509CertificateNames()) == 0 {
		return or, hint, disableHint
	}
	or = append(or, addOwnerToFilter(owner, bson.D{
		{Key: store.AttestationMechanismX509LeadCertificateNameKey, Value: bson.M{
			"$in": queries.GetAttestationMechanismX509CertificateNames(),
		}},
	}))
	if hint == nil && !disableHint {
		if owner != "" {
			hint = &AttestationMechanismX509CertificateNamesOwnerKeyQueryIndex
		} else {
			hint = &AttestationMechanismX509CertificateNamesKeyQueryIndex
		}
	} else {
		hint = nil
		disableHint = true
	}
	return or, hint, disableHint
}

func toEnrollmentGroupFilter(owner string, queries *store.EnrollmentGroupsQuery) (bson.D, *mongo.IndexModel) {
	or, hint := toEnrollmentGroupIDFilter(owner, queries)
	or, hint, disableHint := toEnrollmentGroupHubIDFilter(owner, queries, false, or, hint)
	or, hint, _ = toEnrollmentGroupAttestationMechanismX509CertificateNamesFilter(owner, queries, disableHint, or, hint)

	switch len(or) {
	case 0:
		return addOwnerToFilter(owner, bson.D{}), nil
	case 1:
		return or[0], hint
	}
	return bson.D{{Key: "$or", Value: or}}, hint
}

type DocumentKey struct {
	ID string `bson:"_id"`
}

type StreamEnrollmentGroupEvent struct {
	OperationType string       `bson:"operationType"`
	DocumentKey   *DocumentKey `bson:"documentKey"`
}

type watchIterator struct {
	ctx  context.Context
	iter *mongo.ChangeStream
}

func (i *watchIterator) Next(ctx context.Context) (event store.Event, id string, ok bool) {
	if !i.iter.Next(ctx) {
		return "", "", false
	}
	var v StreamEnrollmentGroupEvent
	if err := i.iter.Decode(&v); err != nil {
		return "", "", false
	}
	if v.DocumentKey != nil {
		id = v.DocumentKey.ID
	}
	switch v.OperationType {
	case "delete":
		event = store.EventDelete
	case "update":
		event = store.EventUpdate
	}
	return event, id, true
}

func (i *watchIterator) Err() error {
	return i.iter.Err()
}

func (i *watchIterator) Close() error {
	return i.iter.Close(i.ctx)
}

func (s *Store) watch(ctx context.Context, col *mongo.Collection) (*watchIterator, error) {
	stream, err := col.Watch(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "operationType", Value: bson.M{"$in": []string{"delete", "update"}}}}}},
	})
	if err != nil {
		return nil, err
	}
	i := watchIterator{
		iter: stream,
		ctx:  ctx,
	}
	return &i, err
}

func (s *Store) WatchEnrollmentGroups(ctx context.Context) (store.WatchEnrollmentGroupIter, error) {
	return s.watch(ctx, s.Collection(enrollmentGroupsCol))
}

func (s *Store) DeleteEnrollmentGroups(ctx context.Context, owner string, query *store.EnrollmentGroupsQuery) (int64, error) {
	opts := options.Delete()
	filter, hint := toEnrollmentGroupFilter(owner, query)
	if hint != nil {
		opts.SetHint(hint.Keys)
	}
	res, err := s.Collection(enrollmentGroupsCol).DeleteMany(ctx, filter, opts)
	if err != nil {
		return -1, fmt.Errorf("cannot remove enrollment groups for owner %v with filter %v: %w", owner, query.GetIdFilter(), err)
	}
	if res.DeletedCount == 0 {
		return -1, fmt.Errorf("cannot remove enrollment groups for owner %v with filter %v: not found", owner, query.GetIdFilter())
	}
	return res.DeletedCount, nil
}

func (s *Store) LoadEnrollmentGroups(ctx context.Context, owner string, query *store.EnrollmentGroupsQuery, h store.LoadEnrollmentGroupsFunc) error {
	col := s.Collection(enrollmentGroupsCol)
	opts := options.Find()
	filter, hint := toEnrollmentGroupFilter(owner, query)
	if hint != nil {
		opts.SetHint(hint.Keys)
	}
	iter, err := col.Find(ctx, filter, opts)
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil
	}
	if err != nil {
		return err
	}

	i := enrollmentGroupsIterator{
		iter: iter,
	}
	err = h(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type enrollmentGroupsIterator struct {
	iter *mongo.Cursor
}

func (i *enrollmentGroupsIterator) Next(ctx context.Context, s *store.EnrollmentGroup) bool {
	if !i.iter.Next(ctx) {
		return false
	}
	err := i.iter.Decode(s)
	return err == nil
}

func (i *enrollmentGroupsIterator) Err() error {
	return i.iter.Err()
}

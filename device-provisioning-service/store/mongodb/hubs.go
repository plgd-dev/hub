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

const hubsCol = "hubs"

func (s *Store) CreateHub(ctx context.Context, owner string, hub *store.Hub) error {
	if err := hub.Validate(owner); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	_, err := s.Collection(hubsCol).InsertOne(ctx, hub)
	return err
}

func (s *Store) updateHub(ctx context.Context, owner string, hub *store.Hub, upsert bool) error {
	if err := hub.Validate(owner); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	res, err := s.Collection(hubsCol).UpdateOne(ctx, toHubFilter(owner, &store.HubsQuery{
		IdFilter: []string{hub.GetId()},
	}), bson.M{"$set": hub}, options.Update().SetUpsert(upsert))
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

func (s *Store) UpsertHub(ctx context.Context, owner string, hub *store.Hub) error {
	return s.updateHub(ctx, owner, hub, true)
}

func (s *Store) UpdateHub(ctx context.Context, owner string, hub *store.Hub) error {
	return s.updateHub(ctx, owner, hub, false)
}

func toHubFilter(owner string, queries *store.HubsQuery) bson.D {
	or := []bson.D{}
	for _, q := range queries.GetIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{{Key: store.IDKey, Value: q}}))
	}
	for _, q := range queries.GetHubIdFilter() {
		or = append(or, addOwnerToFilter(owner, bson.D{{Key: store.HubIDKey, Value: q}}))
	}
	switch len(or) {
	case 0:
		return addOwnerToFilter(owner, bson.D{})
	case 1:
		return or[0]
	}
	return bson.D{{Key: "$or", Value: or}}
}

func (s *Store) DeleteHubs(ctx context.Context, owner string, query *store.HubsQuery) (int64, error) {
	res, err := s.Collection(hubsCol).DeleteMany(ctx, toHubFilter(owner, query))
	if err != nil {
		return -1, fmt.Errorf("cannot remove hubs for owner %v with filter %v: %w", owner, query.GetIdFilter(), err)
	}
	if res.DeletedCount == 0 {
		return -1, fmt.Errorf("cannot remove hubs for owner %v with filter %v: not found", owner, query.GetIdFilter())
	}
	return res.DeletedCount, nil
}

func (s *Store) LoadHubs(ctx context.Context, owner string, query *store.HubsQuery, h store.LoadHubsFunc) error {
	iter, err := s.Collection(hubsCol).Find(ctx, toHubFilter(owner, query))
	if errors.Is(err, mongo.ErrNilDocument) {
		return nil
	}
	if err != nil {
		return err
	}

	i := hubsIterator{
		iter: iter,
	}
	err = h(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type hubsIterator struct {
	iter *mongo.Cursor
}

func (i *hubsIterator) Next(ctx context.Context, s *store.Hub) bool {
	if !i.iter.Next(ctx) {
		return false
	}
	err := i.iter.Decode(s)
	return err == nil
}

func (i *hubsIterator) Err() error {
	return i.iter.Err()
}

func (s *Store) WatchHubs(ctx context.Context) (store.WatchHubIter, error) {
	return s.watch(ctx, s.Collection(hubsCol))
}

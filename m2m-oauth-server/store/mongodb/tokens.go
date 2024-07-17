package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Store) CreateToken(ctx context.Context, owner string, token *pb.Token) (*pb.Token, error) {
	if token.Owner == "" {
		token.Owner = owner
	}
	if token.GetId() == "" {
		token.Id = uuid.NewString()
	}
	if owner != token.GetOwner() {
		return nil, store.ErrInvalidArgument
	}
	err := token.Validate()
	if err != nil {
		return nil, err
	}
	m, err := token.ToBsonMap()
	if err != nil {
		return nil, err
	}
	_, err = s.Store.Collection(tokensCol).InsertOne(ctx, m)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func toFilter(owner string, req *pb.GetTokensRequest) (filter bson.D) {
	if owner != "" {
		filter = append(filter, bson.E{Key: pb.OwnerKey, Value: owner})
	}
	if len(req.GetIdFilter()) > 0 {
		filter = append(filter, bson.E{Key: "_id", Value: bson.M{mongodb.In: req.GetIdFilter()}})
	}
	if len(req.GetAudienceFilter()) > 0 {
		filter = append(filter, bson.E{Key: pb.AudienceKey, Value: bson.M{mongodb.In: req.GetAudienceFilter()}})
	}
	if !req.GetIncludeBlacklisted() {
		filter = append(filter,
			bson.E{
				Key: mongodb.Or, Value: bson.A{
					bson.M{
						pb.BlackListedFlagKey: bson.M{
							mongodb.Exists: false,
						},
					},
					bson.M{
						pb.BlackListedFlagKey: false,
					},
				},
			})
	}
	return filter
}

func processCursor[T any](ctx context.Context, cr *mongo.Cursor, process store.Process[T]) error {
	var errors *multierror.Error
	iter := store.MongoIterator[T]{
		Cursor: cr,
	}
	for {
		var stored T
		if !iter.Next(ctx, &stored) {
			break
		}
		err := process(&stored)
		if err != nil {
			errors = multierror.Append(errors, err)
			break
		}
	}
	errors = multierror.Append(errors, iter.Err())
	errClose := cr.Close(ctx)
	errors = multierror.Append(errors, errClose)
	return errors.ErrorOrNil()
}

func (s *Store) GetTokens(ctx context.Context, owner string, req *pb.GetTokensRequest, process store.ProcessTokens) error {
	if owner == "" {
		return store.ErrInvalidArgument
	}
	filter := toFilter(owner, req)
	cur, err := s.Store.Collection(tokensCol).Find(ctx, filter)
	if err != nil {
		return err
	}
	return processCursor(ctx, cur, process)
}

func (s *Store) DeleteTokens(ctx context.Context, now time.Time) error {
	deleteFilter := bson.D{
		{Key: pb.ExpirationKey, Value: bson.M{"$lt": now.UnixNano()}},
		{Key: pb.ExpirationKey, Value: bson.M{"$gt": int64(0)}},
		{Key: pb.BlackListedFlagKey, Value: true},
	}
	_, err := s.Store.Collection(tokensCol).DeleteMany(ctx, deleteFilter)
	return err
}

func (s *Store) BlacklistTokens(ctx context.Context, owner string, req *pb.BlacklistTokensRequest) (*pb.BlacklistTokensResponse, error) {
	if owner == "" {
		return nil, store.ErrInvalidArgument
	}
	filter := bson.D{
		{Key: pb.OwnerKey, Value: owner},
		{
			Key: mongodb.Or, Value: bson.A{
				bson.M{
					pb.ExpirationKey: bson.M{"$gte": time.Now().UnixNano()},
				},
				bson.M{
					pb.ExpirationKey: bson.M{mongodb.Exists: false},
				},
				bson.M{
					pb.ExpirationKey: int64(0),
				},
			},
		},
		{
			Key: mongodb.Or, Value: bson.A{
				bson.M{pb.BlackListedFlagKey: false},
				bson.M{pb.BlackListedFlagKey: bson.M{mongodb.Exists: false}},
			},
		},
	}
	if len(req.GetIdFilter()) > 0 {
		filter = append(filter, bson.E{Key: "_id", Value: bson.M{mongodb.In: req.GetIdFilter()}})
	}
	blacklisted := pb.Token_BlackListed{
		Flag:      true,
		Timestamp: time.Now().UnixNano(),
	}
	value, err := blacklisted.ToBsonMap()
	if err != nil {
		return nil, err
	}

	update := bson.D{
		{
			Key: mongodb.Set, Value: bson.M{
				pb.BlackListedKey: value,
			},
		},
	}
	ret, err := s.Store.Collection(tokensCol).UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	return &pb.BlacklistTokensResponse{
		Count: ret.MatchedCount,
	}, nil
}

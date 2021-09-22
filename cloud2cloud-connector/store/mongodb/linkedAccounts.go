package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const resLinkedAccountCName = "linkedAccounts"

func validateLinkedAccount(sub store.LinkedAccount) error {
	if sub.ID == "" {
		return fmt.Errorf("invalid ID")
	}
	if sub.UserID == "" {
		return fmt.Errorf("invalid UserID")
	}
	if sub.LinkedCloudID == "" {
		return fmt.Errorf("invalid LinkedCloudID")
	}
	origin := sub.Data.Origin()
	if origin.AccessToken == "" && origin.RefreshToken == "" {
		return fmt.Errorf("invalid Data.OriginCloud.AccessToken and Data.OriginCloud.RefreshToken")
	}
	target := sub.Data.Target()
	if target.AccessToken == "" && target.RefreshToken == "" {
		return fmt.Errorf("invalid Data.TargetCloud.AccessToken and Data.TargetCloud.RefreshToken")
	}
	return nil
}

func (s *Store) InsertLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return fmt.Errorf("cannot insert linked account: %w", err)
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)

	if _, err := col.InsertOne(ctx, sub); err != nil {
		return fmt.Errorf("cannot insert linked account: %w", err)
	}
	return nil
}

func (s *Store) UpdateLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return fmt.Errorf("cannot update linked account: %w", err)
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)
	res, err := col.UpdateOne(ctx, bson.M{"_id": sub.ID}, bson.M{"$set": sub})
	if err != nil {
		return fmt.Errorf("cannot update linked account: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("cannot update linked account: not found")
	}
	return nil
}

func (s *Store) RemoveLinkedAccount(ctx context.Context, linkedAccountId string) error {
	if linkedAccountId == "" {
		return fmt.Errorf("cannot remove linked account: invalid linkedAccountId")
	}
	res, err := s.client.Database(s.DBName()).Collection(resLinkedAccountCName).DeleteOne(ctx, bson.M{"_id": linkedAccountId})
	if err != nil {
		return fmt.Errorf("cannot remove linked account: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("cannot remove linked account: not found")
	}
	return nil
}

func (s *Store) LoadLinkedAccounts(ctx context.Context, query store.Query, h store.LinkedAccountHandler) error {
	var iter *mongo.Cursor
	var err error

	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)
	switch {
	case query.ID != "":
		iter, err = col.Find(ctx, bson.M{"_id": query.ID})
	case query.LinkedCloudID != "":
		iter, err = col.Find(ctx, bson.M{"linkedcloudid": query.LinkedCloudID})
	default:
		iter, err = col.Find(ctx, bson.M{})
	}
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}

	i := iterator{
		iter: iter,
	}
	err = h.Handle(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type iterator struct {
	iter *mongo.Cursor
}

func (i *iterator) Next(ctx context.Context, s *store.LinkedAccount) bool {
	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(s)
	return err == nil
}

func (i *iterator) Err() error {
	return i.iter.Err()
}

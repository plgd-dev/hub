package mongodb

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const resLinkedAccountCName = "linkedAccounts"

func validateLinkedAccount(sub store.LinkedAccount) error {
	if sub.ID == "" {
		return fmt.Errorf("cannot save linked account: invalid ID")
	}
	if sub.UserID == "" {
		return fmt.Errorf("cannot save linked account: invalid UserID")
	}
	if sub.LinkedCloudID == "" {
		return fmt.Errorf("cannot save linked account: invalid LinkedCloudID")
	}
	if sub.TargetCloud.AccessToken == "" && sub.TargetCloud.RefreshToken == "" {
		return fmt.Errorf("cannot save linked account: invalid TargetCloud.AccessToken and TargetCloud.RefreshToken")
	}
	return nil
}

func (s *Store) InsertLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return err
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)

	if _, err := col.InsertOne(ctx, sub); err != nil {
		return fmt.Errorf("cannot insert linked account: %v", err)
	}
	return nil
}

func (s *Store) UpdateLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return err
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)
	if res, err := col.UpdateOne(ctx, bson.M{"_id": sub.ID}, bson.M{"$set": sub}); err != nil {
		return fmt.Errorf("cannot update linked account: %v", err)
	} else {
		if res.MatchedCount == 0 {
			return fmt.Errorf("cannot update linked account: not found")
		}
	}
	return nil
}

func (s *Store) RemoveLinkedAccount(ctx context.Context, linkedAccountId string) error {
	if linkedAccountId == "" {
		return fmt.Errorf("cannot remove linked account: invalid linkedAccountId")
	}
	res, err := s.client.Database(s.DBName()).Collection(resLinkedAccountCName).DeleteOne(ctx, bson.M{"_id": linkedAccountId})
	if err != nil {
		return fmt.Errorf("cannot remove linked account: %v", err)
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
	if err != nil {
		return false
	}

	return true
}

func (i *iterator) Err() error {
	return i.iter.Err()
}

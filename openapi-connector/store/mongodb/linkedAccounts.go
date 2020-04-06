package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const resLinkedAccountCName = "linkedAccounts"

type dbOAuth struct {
	LinkedCloudID string
	AccessToken   string
	RefreshToken  string
	Expiry        int64
}

type dbLinkedAccount struct {
	ID          string `bson:"_id"`
	TargetURL   string
	TargetCloud dbOAuth
	OriginCloud dbOAuth
}

func makeDBLinkedAccount(sub store.LinkedAccount) dbLinkedAccount {
	targetExpiry := int64(0)
	if !sub.TargetCloud.Expiry.IsZero() {
		targetExpiry = sub.TargetCloud.Expiry.UnixNano()
	}
	originExpiry := int64(0)
	if !sub.TargetCloud.Expiry.IsZero() {
		originExpiry = sub.OriginCloud.Expiry.UnixNano()
	}

	return dbLinkedAccount{
		ID:        sub.ID,
		TargetURL: sub.TargetURL,
		TargetCloud: dbOAuth{
			LinkedCloudID: sub.TargetCloud.LinkedCloudID,
			AccessToken:   string(sub.TargetCloud.AccessToken),
			RefreshToken:  sub.TargetCloud.RefreshToken,
			Expiry:        targetExpiry,
		},
		OriginCloud: dbOAuth{
			LinkedCloudID: sub.OriginCloud.LinkedCloudID,
			AccessToken:   string(sub.OriginCloud.AccessToken),
			RefreshToken:  sub.OriginCloud.RefreshToken,
			Expiry:        originExpiry,
		},
	}

}

func validateLinkedAccount(sub store.LinkedAccount) error {
	if sub.ID == "" {
		return fmt.Errorf("cannot save linked account: invalid ID")
	}
	if sub.TargetCloud.LinkedCloudID == "" {
		return fmt.Errorf("cannot save linked account: invalid ConfigId")
	}
	if sub.TargetCloud.AccessToken == "" && sub.TargetCloud.RefreshToken == "" {
		return fmt.Errorf("cannot save linked account: invalid AccessToken and RefreshToken")
	}
	if sub.TargetURL == "" {
		return fmt.Errorf("cannot save linked account: invalid TargetURL")
	}
	return nil
}

func (s *Store) InsertLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return err
	}

	dbSub := makeDBLinkedAccount(sub)
	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)

	if _, err := col.InsertOne(ctx, dbSub); err != nil {
		return fmt.Errorf("cannot insert linked account: %v", err)
	}
	return nil
}

func (s *Store) UpdateLinkedAccount(ctx context.Context, sub store.LinkedAccount) error {
	err := validateLinkedAccount(sub)
	if err != nil {
		return err
	}
	dbSub := makeDBLinkedAccount(sub)
	col := s.client.Database(s.DBName()).Collection(resLinkedAccountCName)

	if res, err := col.UpdateOne(ctx, bson.M{"_id": sub.ID}, bson.M{"$set": dbSub}); err != nil {
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
	default:
		iter, err = col.Find(ctx, bson.M{})
	}
	if err == mongo.ErrNilDocument {
		return nil
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
	var sub dbLinkedAccount

	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}

	s.ID = sub.ID
	s.TargetURL = sub.TargetURL
	s.TargetCloud.LinkedCloudID = sub.TargetCloud.LinkedCloudID
	s.TargetCloud.AccessToken = store.AccessToken(sub.TargetCloud.AccessToken)
	s.TargetCloud.RefreshToken = sub.TargetCloud.RefreshToken
	if sub.TargetCloud.Expiry != 0 {
		s.TargetCloud.Expiry = time.Unix(-1, sub.TargetCloud.Expiry)
	}
	s.OriginCloud.LinkedCloudID = sub.OriginCloud.LinkedCloudID
	s.OriginCloud.AccessToken = store.AccessToken(sub.OriginCloud.AccessToken)
	s.OriginCloud.RefreshToken = sub.OriginCloud.RefreshToken
	if sub.OriginCloud.Expiry != 0 {
		s.OriginCloud.Expiry = time.Unix(-1, sub.OriginCloud.Expiry)
	}

	return true
}

func (i *iterator) Err() error {
	return i.iter.Err()
}

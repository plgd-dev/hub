package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const resLinkedCloudCName = "LinkedCloud"

func validateLinkedCloud(sub store.LinkedCloud) error {
	if sub.ID == "" {
		return fmt.Errorf("cannot save linked cloud: invalid Id")
	}
	if sub.Endpoint.URL == "" {
		return fmt.Errorf("cannot save linked cloud: invalid URL")
	}
	if sub.OAuth.ClientID == "" {
		return fmt.Errorf("cannot save linked cloud: invalid ClientId")
	}
	if sub.OAuth.ClientSecret == "" {
		return fmt.Errorf("cannot save linked cloud: invalid ClientSecret")
	}
	if sub.OAuth.AuthURL == "" {
		return fmt.Errorf("cannot save linked cloud: invalid AuthUrl")
	}
	if sub.OAuth.TokenURL == "" {
		return fmt.Errorf("cannot save linked cloud: invalid TokenUrl")
	}
	return nil
}

func (s *Store) UpdateLinkedCloud(ctx context.Context, sub store.LinkedCloud) error {
	err := validateLinkedCloud(sub)
	if err != nil {
		return err
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedCloudCName)
	res, err := col.UpdateOne(ctx, bson.M{"_id": sub.ID}, bson.M{"$set": sub})
	if err != nil {
		return fmt.Errorf("cannot save linked cloud: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("cannot update linked cloud: not found")
	}
	return nil
}

func (s *Store) InsertLinkedCloud(ctx context.Context, sub store.LinkedCloud) error {
	err := validateLinkedCloud(sub)
	if err != nil {
		return err
	}

	col := s.client.Database(s.DBName()).Collection(resLinkedCloudCName)

	if _, err := col.InsertOne(ctx, sub); err != nil {
		return fmt.Errorf("cannot save linked cloud: %w", err)
	}
	return nil
}

func (s *Store) RemoveLinkedCloud(ctx context.Context, linkedCloudId string) error {
	if linkedCloudId == "" {
		return fmt.Errorf("cannot remove linked cloud: invalid LinkedCloudId")
	}

	res, err := s.client.Database(s.DBName()).Collection(resLinkedCloudCName).DeleteOne(ctx, bson.M{"_id": linkedCloudId})
	if err != nil {
		return fmt.Errorf("cannot remove linked cloud: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("cannot remove linked cloud: not found")
	}
	return nil
}

func (s *Store) LoadLinkedClouds(ctx context.Context, query store.Query, h store.LinkedCloudHandler) error {
	var iter *mongo.Cursor
	var err error
	col := s.client.Database(s.DBName()).Collection(resLinkedCloudCName)
	switch {
	case query.ID != "":
		iter, err = col.Find(ctx, bson.M{"_id": query.ID})
	default:
		iter, err = col.Find(ctx, bson.M{})
	}
	if err == mongo.ErrNilDocument {
		return nil
	}
	if err != nil {
		return err
	}

	i := linkedCloudIterator{
		iter: iter,
	}
	err = h.Handle(ctx, &i)

	errClose := iter.Close(ctx)
	if err == nil {
		return errClose
	}
	return err
}

type linkedCloudIterator struct {
	iter *mongo.Cursor
}

func (i *linkedCloudIterator) Next(ctx context.Context, s *store.LinkedCloud) bool {
	if !i.iter.Next(ctx) {
		return false
	}
	err := i.iter.Decode(s)
	if err != nil {
		return false
	}

	return true
}

func (i *linkedCloudIterator) Err() error {
	return i.iter.Err()
}

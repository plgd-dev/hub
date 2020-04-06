package mongodb

import (
	"context"
	"fmt"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const resLinkedCloudCName = "LinkedCloud"

type dbEndpoint struct {
	AuthUrl  string
	TokenUrl string
}

type dbLinkedCloud struct {
	Id           string `bson:"_id"`
	Name         string
	ClientId     string
	ClientSecret string
	Scopes       []string
	Endpoint     dbEndpoint
	Audience     string
}

func makeDBLinkedCloud(sub store.LinkedCloud) dbLinkedCloud {
	return dbLinkedCloud{
		Id:           sub.ID,
		Name:         sub.Name,
		ClientId:     sub.ClientID,
		ClientSecret: sub.ClientSecret,
		Scopes:       sub.Scopes,
		Audience:     sub.Audience,
		Endpoint: dbEndpoint{
			AuthUrl:  sub.Endpoint.AuthUrl,
			TokenUrl: sub.Endpoint.TokenUrl,
		},
	}

}

func validateLinkedCloud(sub store.LinkedCloud) error {
	if sub.ID == "" {
		return fmt.Errorf("cannot save linked cloud: invalid Id")
	}
	if sub.ClientID == "" {
		return fmt.Errorf("cannot save linked cloud: invalid ClientId")
	}
	if sub.ClientSecret == "" {
		return fmt.Errorf("cannot save linked cloud: invalid ClientSecret")
	}
	if len(sub.Scopes) == 0 {
		return fmt.Errorf("cannot save linked cloud: invalid Scopes")
	}
	if sub.Endpoint.AuthUrl == "" {
		return fmt.Errorf("cannot save linked cloud: invalid AuthUrl")
	}
	if sub.Endpoint.TokenUrl == "" {
		return fmt.Errorf("cannot save linked cloud: invalid TokenUrl")
	}
	return nil
}

func (s *Store) UpdateLinkedCloud(ctx context.Context, sub store.LinkedCloud) error {
	err := validateLinkedCloud(sub)
	if err != nil {
		return err
	}

	dbSub := makeDBLinkedCloud(sub)
	col := s.client.Database(s.DBName()).Collection(resLinkedCloudCName)

	res, err := col.UpdateOne(ctx, bson.M{"_id": sub.ID}, bson.M{"$set": dbSub})
	if err != nil {
		return fmt.Errorf("cannot save linked cloud: %v", err)
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

	dbSub := makeDBLinkedCloud(sub)
	col := s.client.Database(s.DBName()).Collection(resLinkedCloudCName)

	if _, err := col.InsertOne(ctx, dbSub); err != nil {
		return fmt.Errorf("cannot save linked cloud: %v", err)
	}
	return nil
}

func (s *Store) RemoveLinkedCloud(ctx context.Context, LinkedCloudId string) error {
	if LinkedCloudId == "" {
		return fmt.Errorf("cannot remove linked cloud: invalid LinkedCloudId")
	}

	res, err := s.client.Database(s.DBName()).Collection(resLinkedCloudCName).DeleteOne(ctx, bson.M{"_id": LinkedCloudId})
	if err != nil {
		return fmt.Errorf("cannot remove linked cloud: %v", err)
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
	var sub dbLinkedCloud

	if !i.iter.Next(ctx) {
		return false
	}

	err := i.iter.Decode(&sub)
	if err != nil {
		return false
	}
	s.ID = sub.Id
	s.Name = sub.Name
	s.ClientID = sub.ClientId
	s.ClientSecret = sub.ClientSecret
	s.Scopes = sub.Scopes
	s.Audience = sub.Audience
	s.Endpoint = store.Endpoint{
		AuthUrl:  sub.Endpoint.AuthUrl,
		TokenUrl: sub.Endpoint.TokenUrl,
	}

	return true
}

func (i *linkedCloudIterator) Err() error {
	return i.iter.Err()
}

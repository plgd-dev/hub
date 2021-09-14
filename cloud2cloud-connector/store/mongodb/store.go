package mongodb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Store implements an Store for MongoDB.
type Store struct {
	client   *mongo.Client
	dbPrefix string
}

// NewStore creates a new Store.
func NewStore(ctx context.Context, cfg Config, tls *tls.Config) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetTLSConfig(tls))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not ping database client: %w", err)
	}

	return NewStoreWithSession(ctx, client, cfg.Database)
}

// NewStoreWithSession creates a new Store with a session.
func NewStoreWithSession(ctx context.Context, client *mongo.Client, dbPrefix string) (*Store, error) {
	if client == nil {
		return nil, errors.New("no database session")
	}

	if dbPrefix == "" {
		dbPrefix = "default"
	}

	s := &Store{
		client:   client,
		dbPrefix: dbPrefix,
	}

	return s, nil
}

// DBName returns db name
func (s *Store) DBName() string {
	ns := "db"
	return s.dbPrefix + "_" + ns
}

// Clear clears the event storage.
func (s *Store) Clear(ctx context.Context) error {
	var errors []error
	if err := s.client.Database(s.DBName()).Collection(resLinkedAccountCName).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if err := s.client.Database(s.DBName()).Collection(resLinkedCloudCName).Drop(ctx); err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot clear: %v", errors)
	}

	return nil
}

// Close closes the database session.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type OnClearFn = func(context.Context) error

// Store implements an Store for MongoDB.
type Store struct {
	client   *mongo.Client
	dbPrefix string

	onClear OnClearFn
}

// NewStore creates a new Store.
func NewStore(ctx context.Context, cfg Config, tls *tls.Config) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().SetMaxPoolSize(cfg.MaxPoolSize).SetMaxConnIdleTime(cfg.MaxConnIdleTime).ApplyURI(cfg.URI).SetTLSConfig(tls))
	if err != nil {
		return nil, fmt.Errorf("failed to dial database: %w", err)
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping database client: %w", err)
	}

	dbPrefix := cfg.Database
	if dbPrefix == "" {
		dbPrefix = "default"
	}

	s := &Store{
		client:   client,
		dbPrefix: dbPrefix,
	}

	s.onClear = func(c context.Context) error {
		// default clear function drops the whole database
		return s.client.Database(s.DBName()).Drop(ctx)
	}
	return s, nil
}

func NewStoreWithCollection(ctx context.Context, cfg Config, tls *tls.Config, collection string, indexes ...bson.D) (*Store, error) {
	s, err := NewStore(ctx, cfg, tls)
	if err != nil {
		return nil, err
	}
	err = s.EnsureIndex(ctx, collection, indexes...)
	if err != nil {
		errors := []error{
			err,
		}
		if err = s.client.Disconnect(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect mongodb client: %w", err))
		}
		return nil, fmt.Errorf("%v", errors)
	}
	return s, nil
}

func (s *Store) EnsureIndex(ctx context.Context, collection string, indexes ...bson.D) error {
	if len(indexes) == 0 {
		return nil
	}
	subCol := s.client.Database(s.DBName()).Collection(collection)
	if err := ensureIndex(ctx, subCol, indexes...); err != nil {
		return fmt.Errorf("failed to ensure index for collection(%v): %w", collection, err)
	}
	return nil
}

func ensureIndex(ctx context.Context, col *mongo.Collection, indexes ...bson.D) error {
	for _, keys := range indexes {
		opts := &options.IndexOptions{}
		index := mongo.IndexModel{
			Keys:    keys,
			Options: opts,
		}
		_, err := col.Indexes().CreateOne(ctx, index)
		if err != nil {
			if strings.HasPrefix(err.Error(), "(IndexKeySpecsConflict)") {
				//index already exist, just skip error and continue
				continue
			}
			return fmt.Errorf("failed to ensure indexes for collection: %w", err)
		}
	}
	return nil
}

// DBName returns db name
func (s *Store) DBName() string {
	const ns = "db"
	return s.dbPrefix + "_" + ns
}

// Set the function called on Clear
func (s *Store) SetOnClear(onClear OnClearFn) {
	s.onClear = onClear
}

// Clear clears the event storage.
func (s *Store) Clear(ctx context.Context) error {
	if s.onClear != nil {
		return s.onClear(ctx)
	}
	return nil
}

// Get mongodb client
func (s *Store) Client() *mongo.Client {
	return s.client
}

// Close closes the database session.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// Get collection with given name
func (s *Store) Collection(collection string) *mongo.Collection {
	return s.client.Database(s.DBName()).Collection(collection)
}

// Drops the whole database
func (s *Store) DropDatabase(ctx context.Context) error {
	return s.client.Database(s.DBName()).Drop(ctx)
}

// Drops selected collection from database
func (s *Store) DropCollection(ctx context.Context, collection string) error {
	return s.client.Database(s.DBName()).Collection(collection).Drop(ctx)
}

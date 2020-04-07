package mongodb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Store implements an Store for MongoDB.
type Store struct {
	client   *mongo.Client
	dbPrefix string
}

type Config struct {
	Host         string `envconfig:"LINKED_STORE_MONGO_HOST" default:"localhost:27017"`
	DatabaseName string `envconfig:"LINKED_STORE_MONGO_DATABASE" default:"openapiConnector"`
	tlsCfg       *tls.Config
}

// Option provides the means to use function call chaining
type Option func(Config) Config

// WithTLS configures connection to use TLS
func WithTLS(cfg *tls.Config) Option {
	return func(c Config) Config {
		c.tlsCfg = cfg
		return c
	}
}

// NewStore creates a new Store.
func NewStore(ctx context.Context, cfg Config, opts ...Option) (*Store, error) {
	for _, o := range opts {
		cfg = o(cfg)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+cfg.Host).SetTLSConfig(cfg.tlsCfg))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %v", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %v", err)
	}

	return NewStoreWithSession(ctx, client, cfg.DatabaseName)
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

	col := s.client.Database(s.DBName()).Collection(subscriptionCName)

	err := ensureIndex(ctx, col, typeQueryIndex, subscriptionLinkAccountQueryIndex, subscriptionDeviceQueryIndex, subscriptionDeviceHrefQueryIndex)
	if err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("cannot ensure index for device subscription: %v", err)
	}

	return s, nil
}

func ensureIndex(ctx context.Context, col *mongo.Collection, indexes ...bson.D) error {
	for _, keys := range indexes {
		opts := &options.IndexOptions{}
		opts.SetBackground(false)
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
			return fmt.Errorf("cannot ensure indexes for eventstore: %v", err)
		}
	}
	return nil
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
	if err := s.client.Database(s.DBName()).Collection(subscriptionCName).Drop(ctx); err != nil {
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

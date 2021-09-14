package mongodb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

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
	s, err := NewStoreWithSession(ctx, client, cfg.Database)
	if err != nil {
		return nil, err
	}
	return s, nil
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

	subCol := s.client.Database(s.DBName()).Collection(subscriptionsCName)
	err := ensureIndex(ctx, subCol, typeQueryIndex, typeDeviceIDQueryIndex, typeResourceIDQueryIndex, typeInitializedIDQueryIndex)
	if err != nil {
		var errors []error = []error{
			fmt.Errorf("cannot ensure index for device subscription: %w", err),
		}
		if err = client.Disconnect(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to disconnect mongodb client: %w", err))
		}
		return nil, fmt.Errorf("%v", errors)
	}

	return s, nil
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
			return fmt.Errorf("cannot ensure indexes for eventstore: %w", err)
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
	err := s.client.Database(s.DBName()).Collection(subscriptionsCName).Drop(ctx)
	if err != nil {
		return fmt.Errorf("cannot clear: %w", err)
	}
	return nil
}

func incrementSubscriptionSequenceNumber(ctx context.Context, col *mongo.Collection, subscriptionId string) (uint64, error) {
	if subscriptionId == "" {
		return 0, fmt.Errorf("cannot increment sequence number: invalid subscriptionId")
	}

	var res bson.M
	opts := &options.FindOneAndUpdateOptions{}
	result := col.FindOneAndUpdate(ctx, bson.M{"_id": subscriptionId}, bson.M{"$inc": bson.M{sequenceNumberKey: 1}}, opts.SetReturnDocument(options.After))
	if result.Err() != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionId, result.Err())
	}

	err := result.Decode(&res)
	if err != nil {
		return 0, fmt.Errorf("cannot increment sequence number for %v: %w", subscriptionId, err)
	}

	value, ok := res[sequenceNumberKey]
	if !ok {
		return 0, fmt.Errorf("cannot increment sequence number for %v: '%v' not found", subscriptionId, sequenceNumberKey)
	}

	return uint64(value.(int64)) - 1, nil
}

// Close closes the database session.
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

const userDevicesCName = "userdevices"

var userDeviceQueryIndex = bson.D{
	{ownerKey, 1},
	{deviceIDKey, 1},
}

var userDevicesQueryIndex = bson.D{
	{ownerKey, 1},
}

// Store implements an Store for MongoDB.
type Store struct {
	client    *mongo.Client
	dbPrefix  string
	closeFunc []func()
}

// NewStore creates a new Store.
func NewStore(ctx context.Context, cfg Config, logger *zap.Logger) (*Store, error) {
	certManager, err := client.New(cfg.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetTLSConfig(certManager.GetTLSConfig()))
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("could not dial database: %w", err)
	}
	s, err := NewStoreWithSession(ctx, client, cfg.Database)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s.AddCloseFunc(certManager.Close)
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

	subCol := s.client.Database(s.DBName()).Collection(userDevicesCName)
	err := ensureIndex(ctx, subCol, userDeviceQueryIndex, userDevicesQueryIndex)
	if err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("cannot ensure index for device subscription: %w", err)
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
	return s.client.Database(s.DBName()).Drop(ctx)
}

// Close closes the database session.
func (s *Store) Close(ctx context.Context) error {
	err := s.client.Disconnect(ctx)
	for _, f := range s.closeFunc {
		f()
	}
	return err
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Server.
func (s *Store) AddCloseFunc(f func()) {
	s.closeFunc = append(s.closeFunc, f)
}

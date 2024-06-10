package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/integration-service/store"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
}

const integrationCol = "integrationRecords"

func New(ctx context.Context, cfg *Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Store, error) {
	certManager, err := client.New(cfg.Mongo.TLS, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create cert manager: %w", err)
	}
	m, err := pkgMongo.NewStore(ctx, &cfg.Mongo, certManager.GetTLSConfig(), tracerProvider)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s := Store{Store: m}
	s.SetOnClear(func(c context.Context) error {
		return s.clearDatabases(ctx)
	})
	s.AddCloseFunc(certManager.Close)
	return &s, nil
}

func (s *Store) clearDatabases(ctx context.Context) error {
	return s.Collection(integrationCol).Drop(ctx)
}

func (s *Store) Close(ctx context.Context) error {
	return s.Store.Close(ctx)
}

func (s *Store) DeleteExpiredRecords(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

func (s *Store) CreateRecord(ctx context.Context, r *store.ConfigurationRecord) error {

	cl := s.Client()

	col := cl.Database(s.DBName()).Collection(integrationCol)

	var commonNameKeyQueryIndex = mongo.IndexModel{
		Keys: bson.D{
			{Key: "id", Value: 1},
		},
	}

	col.Indexes().CreateOne(ctx, commonNameKeyQueryIndex)

	col.InsertOne(ctx, r)

	return nil
}

func (s *Store) GetRecord(ctx context.Context, confID string, query *store.GetConfigurationRequest, rec *store.ConfigurationRecord) error {

	col := s.Collection(integrationCol)

	filter := bson.D{
		{Key: "id", Value: query.Id},
	}

	iter, err := col.Find(ctx, filter)
	defer iter.Close(context.TODO())

	//send only firts item for now
	for iter.Next(ctx) {
		if err := iter.Decode(&rec); err != nil {
			log.Fatal(err)
		}
	}

	if err := iter.Err(); err != nil {
		log.Fatal(err)
	}

	return err
}

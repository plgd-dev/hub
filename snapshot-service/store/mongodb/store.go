package mongodb

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"go.opentelemetry.io/otel/trace"
)

type Store struct {
	*pkgMongo.Store
}

const snapshotsCol = "snapshots"

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
	return s.Collection(snapshotsCol).Drop(ctx)
}

func (s *Store) Close(ctx context.Context) error {
	return s.Store.Close(ctx)
}

package test

import (
	"context"
	"time"

	storeConfig "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/config"
	storeCqlDB "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/cqldb"
	storeMongo "github.com/plgd-dev/hub/v2/m2m-oauth-server/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func MakeStoreConfig() storeConfig.Config {
	return storeConfig.Config{
		// TODO: add cqldb support
		// Use: config.ACTIVE_DATABASE(),
		CleanUpDeletedTokens:      "0 * * * *",
		ExtendCronParserBySeconds: false,
		Config: database.Config[*storeMongo.Config, *storeCqlDB.Config]{
			Use: database.MongoDB,
			MongoDB: &storeMongo.Config{
				Mongo: mongodb.Config{
					MaxPoolSize:     16,
					MaxConnIdleTime: time.Minute * 4,
					URI:             config.MONGODB_URI,
					Database:        "m2mOAuthServer",
					TLS:             config.MakeTLSClientConfig(),
				},
			},
			CqlDB: &storeCqlDB.Config{
				Embedded: config.MakeCqlDBConfig(),
				Table:    "m2mOAuthServer",
			},
		},
	}
}

func NewMongoStore(t require.TestingT) (*storeMongo.Store, func()) {
	cfg := MakeConfig(t)
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := storeMongo.New(ctx, cfg.Clients.Storage.MongoDB, fileWatcher, logger, noop.NewTracerProvider())
	require.NoError(t, err)

	cleanUp := func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)

		err = fileWatcher.Close()
		require.NoError(t, err)
	}

	return store, cleanUp
}

package test

import (
	"context"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	storeCqlDB "github.com/plgd-dev/hub/v2/snippet-service/store/cqldb"
	storeMongo "github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func HTTPURI(uri string) string {
	return httpTest.HTTPS_SCHEME + config.SNIPPET_SERVICE_HTTP_HOST + uri
}

func MakeHTTPConfig() service.HTTPConfig {
	return service.HTTPConfig{
		Addr:   config.SNIPPET_SERVICE_HTTP_HOST,
		Server: config.MakeHttpServerConfig(),
	}
}

func MakeAPIsConfig() service.APIsConfig {
	grpc := config.MakeGrpcServerConfig(config.SNIPPET_SERVICE_HOST)
	grpc.TLS.ClientCertificateRequired = false
	return service.APIsConfig{
		GRPC: grpc,
		HTTP: MakeHTTPConfig(),
	}
}

func MakeResourceAggregateConfig() service.ResourceAggregateConfig {
	return service.ResourceAggregateConfig{
		Connection: config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST),
	}
}

func MakeClientsConfig() service.ClientsConfig {
	return service.ClientsConfig{
		Storage:                MakeStoreConfig(),
		OpenTelemetryCollector: config.MakeOpenTelemetryCollectorClient(),
		EventBus: service.EventBusConfig{
			NATS:           config.MakeSubscriberConfig(),
			SubscriptionID: "snippet-service",
		},
		ResourceAggregate: MakeResourceAggregateConfig(),
	}
}

func MakeStoreConfig() storeConfig.Config {
	return storeConfig.Config{
		// TODO: add cqldb support
		// Use: config.ACTIVE_DATABASE(),
		CleanUpExpiredUpdates:     "0 * * * *",
		ExtendCronParserBySeconds: false,
		Config: database.Config[*storeMongo.Config, *storeCqlDB.Config]{
			Use: database.MongoDB,
			MongoDB: &storeMongo.Config{
				Mongo: mongodb.Config{
					MaxPoolSize:     16,
					MaxConnIdleTime: time.Minute * 4,
					URI:             config.MONGODB_URI,
					Database:        "snippetService",
					TLS:             config.MakeTLSClientConfig(),
				},
			},
			CqlDB: &storeCqlDB.Config{
				Embedded: config.MakeCqlDBConfig(),
				Table:    "snippets",
			},
		},
	}
}

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.HubID = config.HubID()
	cfg.Log = config.MakeLogConfig(t, "TEST_SNIPPET_SERVICE_LOG_LEVEL", "TEST_SNIPPET_SERVICE_LOG_DUMP_BODY")

	cfg.APIs = MakeAPIsConfig()
	cfg.Clients = MakeClientsConfig()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (*service.Service, func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, cfg service.Config) (*service.Service, func()) {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	return s, func() {
		_ = s.Close()
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

func NewStore(t require.TestingT) (store.Store, func()) {
	cfg := MakeConfig(t)
	if cfg.Clients.Storage.Use == database.CqlDB {
		return nil, nil
	}
	return NewMongoStore(t)
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

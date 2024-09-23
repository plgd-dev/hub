package test

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	storeConfig "github.com/plgd-dev/hub/v2/certificate-authority/store/config"
	storeCqlDB "github.com/plgd-dev/hub/v2/certificate-authority/store/cqldb"
	storeMongo "github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func MakeHTTPConfig() service.HTTPConfig {
	return service.HTTPConfig{
		ExternalAddress: "https://" + config.CERTIFICATE_AUTHORITY_HTTP_HOST,
		Addr:            config.CERTIFICATE_AUTHORITY_HTTP_HOST,
		Server:          config.MakeHttpServerConfig(),
	}
}

func MakeCRLConfig() grpc.CRLConfig {
	if config.ACTIVE_DATABASE() == database.MongoDB {
		return grpc.CRLConfig{
			Enabled:   true,
			ExpiresIn: time.Hour,
		}
	}
	return grpc.CRLConfig{
		Enabled: false,
	}
}

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.HubID = config.HubID()
	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP = MakeHTTPConfig()
	cfg.Signer.CAPool = []urischeme.URIScheme{urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_CERT"))}
	cfg.Signer.KeyFile = urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_KEY"))
	cfg.Signer.CertFile = urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_CERT"))
	cfg.Signer.ValidFrom = "now-1h"
	cfg.Signer.ExpiresIn = time.Hour * 2
	cfg.Signer.CRL = MakeCRLConfig()

	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()
	cfg.Clients.Storage = MakeStorageConfig()

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, cfg service.Config) func() {
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

	return func() {
		_ = s.Close()
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

func MakeStorageConfig() service.StorageConfig {
	return service.StorageConfig{
		CleanUpRecords: "0 1 * * *",
		Embedded: storeConfig.Config{
			Use: config.ACTIVE_DATABASE(),
			MongoDB: &storeMongo.Config{
				Mongo: mongodb.Config{
					MaxPoolSize:     16,
					MaxConnIdleTime: time.Minute * 4,
					URI:             config.MONGODB_URI,
					Database:        "certificateAuthority",
					TLS:             config.MakeTLSClientConfig(),
				},
			},
			CqlDB: &storeCqlDB.Config{
				Embedded: config.MakeCqlDBConfig(),
				Table:    "signedCertificateRecords",
			},
		},
	}
}

func NewCQLStore(t require.TestingT) (*storeCqlDB.Store, func()) {
	cfg := MakeConfig(t)
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := storeCqlDB.New(ctx, cfg.Clients.Storage.Embedded.CqlDB, fileWatcher, logger, noop.NewTracerProvider())
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

func NewStore(t require.TestingT) (store.Store, func()) {
	cfg := MakeConfig(t)
	if cfg.Clients.Storage.Embedded.Use == database.CqlDB {
		return NewCQLStore(t)
	}
	return NewMongoStore(t)
}

func NewMongoStore(t require.TestingT) (*storeMongo.Store, func()) {
	cfg := MakeConfig(t)
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := storeMongo.New(ctx, cfg.Clients.Storage.Embedded.MongoDB, fileWatcher, logger, noop.NewTracerProvider())
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

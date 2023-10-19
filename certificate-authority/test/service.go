package test

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	storeMongo "github.com/plgd-dev/hub/v2/certificate-authority/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.HubID = config.HubID()
	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.APIs.HTTP.Addr = config.CERTIFICATE_AUTHORITY_HTTP_HOST
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	cfg.Signer.CAPool = []urischeme.URIScheme{urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_CERT"))}
	cfg.Signer.KeyFile = urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_KEY"))
	cfg.Signer.CertFile = urischeme.URIScheme(os.Getenv("TEST_ROOT_CA_CERT"))
	cfg.Signer.ValidFrom = "now-1h"
	cfg.Signer.ExpiresIn = time.Hour * 2

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
		MongoDB: storeMongo.Config{
			Mongo: mongodb.Config{
				MaxPoolSize:     16,
				MaxConnIdleTime: time.Minute * 4,
				URI:             config.MONGODB_URI,
				Database:        "certificateAuthority",
				TLS:             config.MakeTLSClientConfig(),
			},
			BulkWrite: storeMongo.BulkWriteConfig{
				Timeout:       time.Minute,
				ThrottleTime:  time.Millisecond * 500,
				DocumentLimit: 1000,
			},
		},
	}
}

func NewMongoStore(t require.TestingT) (*storeMongo.Store, func()) {
	cfg := MakeConfig(t)
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	certManager, err := cmClient.New(cfg.Clients.Storage.MongoDB.Mongo.TLS, fileWatcher, logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := storeMongo.NewStore(ctx, cfg.Clients.Storage.MongoDB, certManager.GetTLSConfig(), logger, trace.NewNoopTracerProvider())
	require.NoError(t, err)

	cleanUp := func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)
		certManager.Close()

		err = fileWatcher.Close()
		require.NoError(t, err)
	}

	return store, cleanUp
}

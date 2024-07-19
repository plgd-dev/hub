package test

import (
	"github.com/plgd-dev/hub/v2/identity-store/persistence/cqldb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence/mongodb"
	"github.com/plgd-dev/hub/v2/identity-store/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.HubID = config.HubID()

	cfg.Log = config.MakeLogConfig(t, "TEST_IDENTITY_STORE_LOG_LEVEL", "TEST_IDENTITY_STORE_LOG_DUMP_BODY")

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.IDENTITY_STORE_HOST)

	cfg.Clients.Storage.Use = config.ACTIVE_DATABASE()
	cfg.Clients.Storage.MongoDB = &mongodb.Config{}
	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	cfg.Clients.Storage.MongoDB.Database = config.IDENTITY_STORE_DB
	cfg.Clients.Storage.CqlDB = &cqldb.Config{}
	cfg.Clients.Storage.CqlDB.Embedded = config.MakeCqlDBConfig()

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig(t)
	cfg.Clients.OpenTelemetryCollector = config.MakeOpenTelemetryCollectorClient()

	err := cfg.Validate()
	require.NoError(t, err)
	return cfg
}

package test

import (
	"testing"

	"github.com/plgd-dev/hub/identity-store/service"
	"github.com/plgd-dev/hub/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.IDENTITY_STORE_HOST)

	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	cfg.Clients.Storage.MongoDB.Database = config.IDENTITY_STORE_DB

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig()

	err := cfg.Validate()
	require.NoError(t, err)
	return cfg
}

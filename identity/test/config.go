package test

import (
	"testing"

	"github.com/plgd-dev/cloud/identity/service"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.IDENTITY_HOST)

	cfg.Clients.Storage.OwnerClaim = config.OWNER_CLAIM
	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	cfg.Clients.Storage.MongoDB.Database = "ownersDevices"

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig()

	err := cfg.Validate()
	require.NoError(t, err)
	return cfg
}

package jetstream_test

import (
	"context"
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/jetstream"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestEventStore(t *testing.T) {
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	host := os.Getenv("NATS_SERVER_HOST")
	if host == "" {
		host = "nats://localhost:4223"
	}
	var cfg jetstream.Config
	err = envconfig.Process("", &cfg)
	assert.NoError(t, err)

	ctx := context.Background()
	cfg.URL = host
	cfg.Options = []nats.Option{nats.Secure(tlsConfig)}

	store, err := jetstream.NewEventStore(
		cfg,
		func(f func()) error { go f(); return nil },
		jetstream.WithMarshaler(bson.Marshal),
		jetstream.WithUnmarshaler(bson.Unmarshal),
	)
	require.NoError(t, err)
	require.NotNil(t, store)

	defer store.Close(ctx)
	defer func() {
		t.Log("clearing db")
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	t.Log("event store with default namespace")
	AcceptanceTest(t, ctx, store)
}

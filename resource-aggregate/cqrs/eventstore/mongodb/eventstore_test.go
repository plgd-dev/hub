package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestEventStore(t *testing.T) {
	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)

	ctx := context.Background()

	store, err := mongodb.New(
		ctx,
		mongodb.Config{
			URI: "mongodb://localhost:27017",
			TLS: config.MakeTLSClientConfig(),
		},
		logger,
		mongodb.WithMarshaler(bson.Marshal),
		mongodb.WithUnmarshaler(bson.Unmarshal),
	)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer store.Close(ctx)
	defer func() {
		t.Log("clearing db")
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	t.Log("event store with default namespace")
	test.AcceptanceTest(t, ctx, store)

	t.Log("clearing db")
	store.Clear(ctx)
	test.GetEventsTest(t, ctx, store)
}

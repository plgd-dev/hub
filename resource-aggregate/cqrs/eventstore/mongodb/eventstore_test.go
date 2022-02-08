package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func NewTestEventStore(ctx context.Context, logger log.Logger) (*mongodb.EventStore, error) {
	store, err := mongodb.New(
		ctx,
		mongodb.Config{
			Embedded: pkgMongo.Config{
				URI: "mongodb://localhost:27017",
				TLS: config.MakeTLSClientConfig(),
			},
		},
		logger,
		mongodb.WithMarshaler(bson.Marshal),
		mongodb.WithUnmarshaler(bson.Unmarshal),
	)
	return store, err
}

func TestEventStore(t *testing.T) {
	logger, err := log.NewLogger(log.MakeDefaultConfig())
	require.NoError(t, err)

	ctx := context.Background()
	store, err := NewTestEventStore(ctx, logger)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)
	}()

	t.Log("event store with default namespace")
	test.AcceptanceTest(t, ctx, store)

	t.Log("clearing collections")
	err = store.ClearCollections(ctx)
	require.NoError(t, err)
	test.GetEventsTest(t, ctx, store)
}

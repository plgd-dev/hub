package mongodb_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/trace"
)

func NewTestEventStore(ctx context.Context, fileWatcher *fsnotify.Watcher, logger log.Logger) (*mongodb.EventStore, error) {
	store, err := mongodb.New(
		ctx,
		&mongodb.Config{
			Embedded: pkgMongo.Config{
				URI: "mongodb://localhost:27017",
				TLS: config.MakeTLSClientConfig(),
			},
		},
		fileWatcher,
		logger,
		trace.NewNoopTracerProvider(),
		mongodb.WithMarshaler(bson.Marshal),
		mongodb.WithUnmarshaler(bson.Unmarshal),
	)
	return store, err
}

func TestEventStore(t *testing.T) {
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	ctx := context.Background()
	store, err := NewTestEventStore(ctx, fileWatcher, logger)
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	t.Log("event store with default namespace")
	AcceptanceTest(ctx, t, store)

	t.Log("clearing collections")
	err = store.ClearCollections(ctx)
	require.NoError(t, err)
	GetEventsTest(ctx, t, store)
}

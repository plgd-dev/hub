package cqldb_test

import (
	"context"
	"testing"

	pkgCqldb "github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/trace"
)

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
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() {
		t.Log("clearing db")
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	t.Log("event store with default namespace")
	test.AcceptanceTest(ctx, t, store)

	t.Log("clearing collections")
	err = store.ClearTable(ctx)
	require.NoError(t, err)
	test.GetEventsTest(ctx, t, store)
}

func NewTestEventStore(ctx context.Context, fileWatcher *fsnotify.Watcher, logger log.Logger) (*cqldb.EventStore, error) {
	store, err := cqldb.New(
		ctx,
		&cqldb.Config{
			Embedded: pkgCqldb.Config{
				Hosts:    config.SCYLLA_HOSTS,
				TLS:      config.MakeTLSClientConfig(),
				NumConns: 1,
				Keyspace: pkgCqldb.KeyspaceConfig{
					Name:   "example",
					Create: true,
					Replication: map[string]interface{}{
						"class":              "SimpleStrategy",
						"replication_factor": 1,
					},
				},
			},
			Table: "test",
		},
		fileWatcher,
		logger,
		trace.NewNoopTracerProvider(),
		cqldb.WithMarshaler(bson.Marshal),
		cqldb.WithUnmarshaler(bson.Unmarshal),
	)
	return store, err
}

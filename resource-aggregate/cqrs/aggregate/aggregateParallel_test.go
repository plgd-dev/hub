package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	eventstoreConfig "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/config"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/atomic"
)

func testNewEventstore(ctx context.Context, t *testing.T) (eventstore.EventStore, func()) {
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	config := eventstoreConfig.Config{
		Use:     config.ACTIVE_DATABASE(),
		MongoDB: config.MakeEventsStoreMongoDBConfig(),
		CqlDB:   config.MakeEventsStoreCqlDBConfig(),
	}
	switch config.Use {
	case database.MongoDB:
		s, err := mongodb.New(ctx, config.MongoDB, fileWatcher, logger, noop.NewTracerProvider())
		require.NoError(t, err)
		return s, func() {
			errC := s.Clear(ctx)
			require.NoError(t, errC)
			errC = s.Close(ctx)
			require.NoError(t, errC)
			errC = fileWatcher.Close()
			require.NoError(t, errC)
		}
	case database.CqlDB:
		s, err := cqldb.New(ctx, config.CqlDB, fileWatcher, logger, noop.NewTracerProvider())
		require.NoError(t, err)
		return s, func() {
			errC := s.Clear(ctx)
			require.NoError(t, errC)
			errC = s.Close(ctx)
			require.NoError(t, errC)
			errC = fileWatcher.Close()
			require.NoError(t, errC)
		}
	}
	require.NoError(t, fmt.Errorf("invalid eventstore use('%v')", config.Use))
	return nil, func() {}
}

func cleanUpToSnapshot(ctx context.Context, t *testing.T, store eventstore.EventStore, evs []eventstore.Event) {
	for _, event := range evs {
		if err := store.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: event.GroupID(), AggregateID: event.AggregateID(), Version: event.Version()}}); err != nil && !errors.Is(err, eventstore.ErrNotSupported) {
			assert.NoError(t, err)
		}
		fmt.Printf("snapshot at version %v\n", event.Version())
	}
}

func TestParallelRequest(t *testing.T) {
	ctx := context.Background()
	store, tearDown := testNewEventstore(ctx, t)
	defer func() {
		tearDown()
	}()

	deviceID := "7397398d-3ae8-4d9a-62d6-511f7b736a60"
	href := "/test/resource/1"

	newAggregate := func(deviceID, href string) *aggregate.Aggregate {
		a, err := aggregate.NewAggregate(deviceID, commands.NewResourceID(deviceID, href).ToUUID().String(), aggregate.NewDefaultRetryFunc(128), store, func(context.Context, string, string) (aggregate.AggregateModel, error) {
			ev := events.NewResourceStateSnapshotTakenForCommand("test", "test", "hubID", nil)
			ev.ResourceId = commands.NewResourceID(deviceID, href)
			return ev, nil
		}, nil)
		require.NoError(t, err)
		return a
	}

	numParallel := 3
	var wg sync.WaitGroup
	var anyError atomic.Error
	for i := range numParallel {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 1000 {
				if anyError.Load() != nil {
					return
				}
				commandContentChanged := commands.NotifyResourceChangedRequest{
					ResourceId: commands.NewResourceID(deviceID, href),
					Content: &commands.Content{
						Data:        []byte("hello world" + fmt.Sprintf("%v.%v", id, j)),
						ContentType: "text",
					},
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: uuid.New().String(),
					},
					Status: commands.Status_OK,
				}
				aggr := newAggregate(commandContentChanged.GetResourceId().GetDeviceId(), commandContentChanged.GetResourceId().GetHref())
				events, err := aggr.HandleCommand(ctx, &commandContentChanged)
				if err != nil {
					anyError.Store(err)
					return
				}
				cleanUpToSnapshot(ctx, t, store, events)
			}
		}(i)
	}
	wg.Wait()
	err := anyError.Load()
	require.NoError(t, err)
}

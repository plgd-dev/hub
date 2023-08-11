package aggregate_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

func testNewEventstore(ctx context.Context, t *testing.T) *mongodb.EventStore {
	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	cfg := config.MakeEventsStoreMongoDBConfig()
	store, err := mongodb.New(
		ctx,
		cfg,
		fileWatcher,
		logger,
		trace.NewNoopTracerProvider(),
	)
	require.NoError(t, err)
	require.NotNil(t, store)

	return store
}

func cleanUpToSnapshot(ctx context.Context, t *testing.T, store *mongodb.EventStore, evs []eventstore.Event) {
	for _, event := range evs {
		if event.IsSnapshot() {
			if err := store.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: event.GroupID(), AggregateID: event.AggregateID(), Version: event.Version()}}); err != nil {
				require.NoError(t, err)
			}
			fmt.Printf("snapshot at version %v\n", event.Version())
			break
		}
	}
}

func Test_parallelRequest(t *testing.T) {
	ctx := context.Background()
	token := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": "test",
	})
	ctx = events.CtxWithHubID(grpc.CtxWithIncomingToken(ctx, token), "hubID")
	store := testNewEventstore(ctx, t)
	defer func() {
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	deviceID := "7397398d-3ae8-4d9a-62d6-511f7b736a60"
	href := "/test/resource/1"

	newAggregate := func(deviceID, href string) *aggregate.Aggregate {
		a, err := aggregate.NewAggregate(deviceID, commands.NewResourceID(deviceID, href).ToUUID().String(), aggregate.NewDefaultRetryFunc(64), 16, store, func(context.Context) (aggregate.AggregateModel, error) {
			ev := events.NewResourceStateSnapshotTaken()
			ev.ResourceId = commands.NewResourceID(deviceID, href)
			return ev, nil
		}, nil)
		require.NoError(t, err)
		return a
	}

	numParallel := 3
	var wg sync.WaitGroup
	var anyError atomic.Error
	for i := 0; i < numParallel; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
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

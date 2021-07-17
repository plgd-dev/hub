package aggregate_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func testNewEventstore(ctx context.Context, t *testing.T) *mongodb.EventStore {
	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)
	store, err := mongodb.New(
		ctx,
		config.MakeEventsStoreMongoDBConfig(),
		logger,
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

//old 452.969s
//new 474.906s
func Test_parallelRequest(t *testing.T) {
	ctx := context.Background()
	ctx = grpc.CtxWithIncomingOwner(ctx, "test")
	store := testNewEventstore(ctx, t)
	defer store.Close(ctx)
	defer func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	deviceID := "7397398d-3ae8-4d9a-62d6-511f7b736a60"
	href := "/test/resource/1"

	newAggragate := func(deviceID, href string) *aggregate.Aggregate {
		a, err := aggregate.NewAggregate(deviceID, commands.NewResourceID(deviceID, href).ToUUID(), aggregate.NewDefaultRetryFunc(64), 16, store, func(context.Context) (aggregate.AggregateModel, error) {
			ev := events.NewResourceStateSnapshotTaken()
			ev.ResourceId = commands.NewResourceID(deviceID, href)
			return ev, nil
		}, nil)
		require.NoError(t, err)
		return a
	}

	numParallel := 3
	var wg sync.WaitGroup
	var anyError atomic.Bool
	for i := 0; i < numParallel; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100000; j++ {
				if anyError.Load() {
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
				aggr := newAggragate(commandContentChanged.GetResourceId().GetDeviceId(), commandContentChanged.GetResourceId().GetHref())
				events, err := aggr.HandleCommand(ctx, &commandContentChanged)
				if err != nil {
					anyError.Store(true)
					require.NoError(t, err)
					return
				}
				cleanUpToSnapshot(ctx, t, store, events)
			}
		}(i)
	}
	wg.Wait()
}

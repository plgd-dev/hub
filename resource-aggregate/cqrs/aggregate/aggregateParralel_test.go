package aggregate_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func testNewEventstore(t *testing.T) *mongodb.EventStore {
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	store, err := mongodb.NewEventStore(
		mongodb.Config{
			URI: "mongodb://localhost:27017",
		},
		func(f func()) error { go f(); return nil },
		mongodb.WithTLS(tlsConfig),
	)
	require.NoError(t, err)
	require.NotNil(t, store)

	return store
}

func cleanUpToSnapshot(ctx context.Context, t *testing.T, store *mongodb.EventStore, evs []eventstore.Event) {
	for _, event := range evs {
		if ru, ok := event.(*events.ResourceStateSnapshotTaken); ok {
			if err := store.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: ru.GroupId(), AggregateID: ru.AggregateId(), Version: ru.Version()}}); err != nil {
				require.NoError(t, err)
			}
			fmt.Printf("snapshot at version %v\n", event.Version())
			break
		}
	}
}

func Test_parallelRequest(t *testing.T) {
	store := testNewEventstore(t)
	ctx := context.Background()
	ctx = grpc.CtxWithIncomingUserID(ctx, "test")
	defer store.Close(ctx)
	defer func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
	}()

	deviceID := "7397398d-3ae8-4d9a-62d6-511f7b736a60"
	href := "/test/resource/1"
	commandPub := pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Resource: &pb.Resource{
			Id:       utils.MakeResourceId(deviceID, href),
			DeviceId: deviceID,
		},
		CommandMetadata:      &pb.CommandMetadata{},
		AuthorizationContext: &pb.AuthorizationContext{},
	}

	newAggragate := func(deviceID, href string) *aggregate.Aggregate {
		a, err := aggregate.NewAggregate(deviceID, utils.MakeResourceId(deviceID, href), aggregate.NewDefaultRetryFunc(32), 16, store, func(context.Context) (aggregate.AggregateModel, error) {
			ev := events.NewResourceStateSnapshotTaken()
			ev.Id = utils.MakeResourceId(deviceID, href)
			return ev, nil
		}, nil)
		require.NoError(t, err)
		return a
	}

	concurrencyExcepTestA := newAggragate(commandPub.GetResourceId().GetDeviceId(), commandPub.GetResourceId().GetHref())
	_, err := concurrencyExcepTestA.HandleCommand(ctx, &commandPub)
	require.NoError(t, err)

	numParallel := 3
	var wg sync.WaitGroup
	var anyError atomic.Bool
	for i := 0; i < numParallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100000; j++ {
				if anyError.Load() {
					return
				}
				commandContentChanged := pb.NotifyResourceChangedRequest{
					ResourceId: &pb.ResourceId{
						DeviceId: deviceID,
						Href:     href,
					},
					Content: &pb.Content{
						Data:        []byte("hello world"),
						ContentType: "text",
					},
					CommandMetadata: &pb.CommandMetadata{
						ConnectionId: uuid.New().String(),
					},
					Status:               pb.Status_OK,
					AuthorizationContext: &pb.AuthorizationContext{},
				}
				aggr := newAggragate(commandPub.GetResourceId().GetDeviceId(), commandPub.GetResourceId().GetHref())
				events, err := aggr.HandleCommand(ctx, &commandContentChanged)
				if err != nil {
					anyError.Store(true)
					require.NoError(t, err)
					return
				}
				cleanUpToSnapshot(ctx, t, store, events)
			}
		}()
	}
	wg.Wait()
}

package projection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/assert"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/stretchr/testify/require"
)

type mockEventHandler struct {
	pb []eventstore.EventUnmarshaler
}

func (eh *mockEventHandler) Handle(ctx context.Context, iter eventstore.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}
		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		eh.pb = append(eh.pb, eu)
	}
	return nil
}

func (eh *mockEventHandler) SnapshotEventType() string {
	var rs events.ResourceStateSnapshotTaken
	return rs.SnapshotEventType()
}

func TestProjection(t *testing.T) {
	numEventsInSnapshot := 1
	waitForSubscription := time.Second * 1

	topics := []string{"test_projection_topic0_" + uuid.Must(uuid.NewV4()).String(), "test_projection_topic1_" + uuid.Must(uuid.NewV4()).String()}
	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)

	publisher, err := publisher.New(config.MakePublisherConfig(), logger)
	require.NoError(t, err)
	assert.NotNil(t, publisher)
	defer publisher.Close()

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	subscriber, err := subscriber.New(config.MakeSubscriberConfig(),
		logger,
		subscriber.WithGoPool(pool.Submit),
	)
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)
	defer subscriber.Close()

	ctx := context.Background()
	ctx = kitNetGrpc.CtxWithIncomingOwner(ctx, "test")

	store, err := mongodb.New(
		ctx,
		config.MakeEventsStoreMongoDBConfig(),
		logger,
		mongodb.WithGoPool(pool.Submit),
	)

	require.NoError(t, err)
	require.NotNil(t, store)

	defer store.Close(ctx)
	defer func() {
		err = store.Clear(ctx)
		require.NoError(t, err)
	}()

	res1 := commands.ResourceId{
		DeviceId: "1",
		Href:     "ID1",
	}

	res2 := commands.ResourceId{
		DeviceId: "1",
		Href:     "ID2",
	}

	res3 := commands.ResourceId{
		DeviceId: "1",
		Href:     "ID3",
	}

	commandPub1 := commands.NotifyResourceChangedRequest{
		ResourceId:      &res1,
		Content:         &commands.Content{Data: []byte("asd")},
		CommandMetadata: &commands.CommandMetadata{},
	}

	commandPub2 := commands.NotifyResourceChangedRequest{
		ResourceId:      &res2,
		Content:         &commands.Content{Data: []byte("asd")},
		CommandMetadata: &commands.CommandMetadata{},
	}

	commandPub3 := commands.NotifyResourceChangedRequest{
		ResourceId:      &res3,
		Content:         &commands.Content{Data: []byte("asd")},
		CommandMetadata: &commands.CommandMetadata{},
	}

	a1, err := aggregate.NewAggregate(res1.DeviceId, res1.ToUUID(), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{
			ResourceId:    &res1,
			EventMetadata: &events.EventMetadata{},
		}, nil
	}, nil)
	require.NoError(t, err)

	evs, err := a1.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, evs)

	snapshotEventType := func() string {
		s := &events.ResourceStateSnapshotTaken{}
		return s.SnapshotEventType()
	}

	a2, err := aggregate.NewAggregate(res2.DeviceId, res2.ToUUID(), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{
			ResourceId:    &res2,
			EventMetadata: &events.EventMetadata{},
		}, nil
	}, nil)
	require.NoError(t, err)

	evs, err = a2.HandleCommand(ctx, &commandPub2)
	require.NoError(t, err)
	require.NotNil(t, evs)

	projection, err := newProjection(ctx, store, "testProjection", subscriber, func(context.Context, string, string) (eventstore.Model, error) { return &mockEventHandler{}, nil }, nil)
	require.NoError(t, err)

	err = projection.Project(ctx, []eventstore.SnapshotQuery{{
		GroupID:           res1.DeviceId,
		AggregateID:       res1.ToUUID(),
		SnapshotEventType: snapshotEventType(),
	}})
	require.NoError(t, err)
	require.Equal(t, 1, len(projection.Models(nil)))

	err = projection.Project(ctx, []eventstore.SnapshotQuery{{
		GroupID:           res2.DeviceId,
		AggregateID:       res2.ToUUID(),
		SnapshotEventType: snapshotEventType(),
	}})
	require.NoError(t, err)
	require.Equal(t, 2, len(projection.Models(nil)))

	err = projection.SubscribeTo(topics)
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	a3, err := aggregate.NewAggregate(res3.DeviceId, res3.ToUUID(), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{
			ResourceId:    &res3,
			EventMetadata: &events.EventMetadata{},
		}, nil
	}, nil)

	evs, err = a3.HandleCommand(ctx, &commandPub3)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, res3.DeviceId, res3.ToUUID(), e)
		require.NoError(t, err)
	}
	time.Sleep(time.Second)

	require.Equal(t, 3, len(projection.Models(nil)))

	err = projection.SubscribeTo(topics[0:1])
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	err = projection.Forget([]eventstore.SnapshotQuery{{
		GroupID:           res3.DeviceId,
		AggregateID:       res3.ToUUID(),
		SnapshotEventType: snapshotEventType(),
	}})
	require.NoError(t, err)

	time.Sleep(time.Second)
	projection.lock.Lock()
	require.Equal(t, 2, len(projection.Models(nil)))
	projection.lock.Unlock()

	err = projection.SubscribeTo(nil)
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	evs, err = a1.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, res1.DeviceId, res1.ToUUID(), e)
		require.NoError(t, err)
	}

	projection.lock.Lock()
	require.Equal(t, 2, len(projection.Models(nil)))
	projection.lock.Unlock()
}

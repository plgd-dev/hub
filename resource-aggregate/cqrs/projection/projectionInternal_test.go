package projection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
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

func TestProjection(t *testing.T) {
	waitForSubscription := time.Second * 1

	topics := []string{"test_projection_topic0_" + uuid.Must(uuid.NewRandom()).String(), "test_projection_topic1_" + uuid.Must(uuid.NewRandom()).String()}
	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	naPubClient, publisher, err := natsTest.NewClientAndPublisher(config.MakePublisherConfig(), fileWatcher, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	assert.NotNil(t, publisher)
	defer func() {
		publisher.Close()
		naPubClient.Close()
	}()

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	naSubClient, subscriber, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher,
		logger,
		subscriber.WithGoPool(pool.Submit),
		subscriber.WithUnmarshaler(utils.Unmarshal),
	)
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)
	defer func() {
		subscriber.Close()
		naSubClient.Close()
	}()

	ctx := context.Background()

	store, err := mongodb.New(
		ctx,
		config.MakeEventsStoreMongoDBConfig(),
		fileWatcher,
		logger,
		noop.NewTracerProvider(),
	)

	require.NoError(t, err)
	require.NotNil(t, store)

	defer func() {
		errC := store.Clear(ctx)
		require.NoError(t, errC)
		_ = store.Close(ctx)
	}()

	res1 := commands.ResourceId{
		DeviceId: test.GenerateDeviceIDbyIdx(1),
		Href:     "ID1",
	}

	res2 := commands.ResourceId{
		DeviceId: test.GenerateDeviceIDbyIdx(1),
		Href:     "ID2",
	}

	res3 := commands.ResourceId{
		DeviceId: test.GenerateDeviceIDbyIdx(1),
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

	a1, err := aggregate.NewAggregate(res1.DeviceId, res1.ToUUID().String(), aggregate.NewDefaultRetryFunc(1), store, func(context.Context) (aggregate.AggregateModel, error) {
		s := events.NewResourceStateSnapshotTakenForCommand("test", "test", "hubID")
		s.ResourceId = &res1
		return s, nil
	}, nil)
	require.NoError(t, err)

	evs, err := a1.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, evs)

	a2, err := aggregate.NewAggregate(res2.DeviceId, res2.ToUUID().String(), aggregate.NewDefaultRetryFunc(1), store, func(context.Context) (aggregate.AggregateModel, error) {
		s := events.NewResourceStateSnapshotTakenForCommand("test", "test", "hubID")
		s.ResourceId = &res2
		return s, nil
	}, nil)
	require.NoError(t, err)

	evs, err = a2.HandleCommand(ctx, &commandPub2)
	require.NoError(t, err)
	require.NotNil(t, evs)

	projection, err := newProjection(ctx, store, "testProjection", subscriber, func(context.Context, string, string) (eventstore.Model, error) { return &mockEventHandler{}, nil }, nil)
	require.NoError(t, err)

	err = projection.Project(ctx, []eventstore.SnapshotQuery{{
		GroupID:     res1.DeviceId,
		AggregateID: res1.ToUUID().String(),
	}})
	require.NoError(t, err)
	models := []eventstore.Model{}
	projection.Models(nil, func(m eventstore.Model) (wantNext bool) {
		models = append(models, m)
		return true
	})
	require.Equal(t, 1, len(models))

	err = projection.Project(ctx, []eventstore.SnapshotQuery{{
		GroupID:     res2.DeviceId,
		AggregateID: res2.ToUUID().String(),
	}})
	require.NoError(t, err)
	models = []eventstore.Model{}
	projection.Models(nil, func(m eventstore.Model) (wantNext bool) {
		models = append(models, m)
		return true
	})
	require.Equal(t, 2, len(models))

	err = projection.SubscribeTo(topics)
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	a3, err := aggregate.NewAggregate(res3.DeviceId, res3.ToUUID().String(), aggregate.NewDefaultRetryFunc(1), store, func(context.Context) (aggregate.AggregateModel, error) {
		s := events.NewResourceStateSnapshotTakenForCommand("test", "test", "hubID")
		s.ResourceId = &res3
		return s, nil
	}, nil)
	require.NoError(t, err)

	evs, err = a3.HandleCommand(ctx, &commandPub3)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, res3.DeviceId, res3.ToUUID().String(), e)
		require.NoError(t, err)
	}
	time.Sleep(time.Second)

	models = []eventstore.Model{}
	projection.Models(nil, func(m eventstore.Model) (wantNext bool) {
		models = append(models, m)
		return true
	})
	require.Equal(t, 3, len(models))

	err = projection.SubscribeTo(topics[0:1])
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	err = projection.Forget([]eventstore.SnapshotQuery{{
		GroupID:     res3.DeviceId,
		AggregateID: res3.ToUUID().String(),
	}})
	require.NoError(t, err)

	time.Sleep(time.Second)
	projection.lock.Lock()

	models = []eventstore.Model{}
	projection.Models(nil, func(m eventstore.Model) (wantNext bool) {
		models = append(models, m)
		return true
	})
	require.Equal(t, 2, len(models))
	projection.lock.Unlock()

	err = projection.SubscribeTo(nil)
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	commandPub1.Content.Data = []byte("commandPub1.Content.Data")
	commandPub1.CommandMetadata.Sequence++
	evs, err = a1.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, res1.DeviceId, res1.ToUUID().String(), e)
		require.NoError(t, err)
	}

	projection.lock.Lock()
	models = []eventstore.Model{}
	projection.Models(nil, func(m eventstore.Model) (wantNext bool) {
		models = append(models, m)
		return true
	})
	require.Equal(t, 2, len(models))
	projection.lock.Unlock()
}

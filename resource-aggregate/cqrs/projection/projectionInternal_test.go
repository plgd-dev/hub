package projection

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"

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
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	publisher, err := nats.NewPublisher(nats.Config{
		URL: "nats://localhost:4222",
	}, nats.WithTLS(tlsConfig))
	require.NoError(t, err)
	assert.NotNil(t, publisher)
	defer publisher.Close()

	subscriber, err := nats.NewSubscriber(nats.Config{
		URL: "nats://localhost:4222",
	},
		func(f func()) error { go f(); return nil },
		func(err error) {
			assert.NoError(t, err)
		},
		nats.WithTLS(tlsConfig),
	)
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)
	defer subscriber.Close()

	// Local Mongo testing with Docker
	host := os.Getenv("MONGO_HOST")

	if host == "" {
		// Default to localhost
		host = "localhost:27017"
	}

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	ctx := context.Background()
	ctx = kitNetGrpc.CtxWithIncomingUserID(ctx, "test")

	store, err := mongodb.NewEventStore(
		mongodb.Config{
			URI: "mongodb://localhost:27017",
		},
		func(f func()) error { go f(); return nil },
		mongodb.WithTLS(tlsConfig),
	)
	/*bson.Marshal, bson.Unmarshal*/
	require.NoError(t, err)
	require.NotNil(t, store)

	defer store.Close(ctx)
	defer func() {
		err = store.Clear(ctx)
		require.NoError(t, err)
	}()

	type Path struct {
		DeviceID string
		Href     string
	}

	path1 := Path{
		DeviceID: "1",
		Href:     "ID1",
	}

	path2 := Path{
		DeviceID: "1",
		Href:     "ID2",
	}

	path3 := Path{
		DeviceID: "1",
		Href:     "ID3",
	}

	commandPub1 := pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: path1.DeviceID,
			Href:     path1.Href,
		},
		Resource: &pb.Resource{
			Id:       utils.MakeResourceId(path1.DeviceID, path1.Href),
			DeviceId: path1.DeviceID,
		},
		AuthorizationContext: &pb.AuthorizationContext{},
		CommandMetadata:      &pb.CommandMetadata{},
	}

	commandPub2 := pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: path2.DeviceID,
			Href:     path2.Href,
		},
		Resource: &pb.Resource{
			Id:       utils.MakeResourceId(path2.DeviceID, path2.Href),
			DeviceId: path2.DeviceID,
		},
		AuthorizationContext: &pb.AuthorizationContext{},
		CommandMetadata:      &pb.CommandMetadata{},
	}

	commandPub3 := pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: path3.DeviceID,
			Href:     path3.Href,
		},
		Resource: &pb.Resource{
			Id:       utils.MakeResourceId(path3.DeviceID, path3.Href),
			DeviceId: path3.DeviceID,
		},
		AuthorizationContext: &pb.AuthorizationContext{},
		CommandMetadata:      &pb.CommandMetadata{},
	}

	commandUnpub1 := pb.UnpublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: path1.DeviceID,
			Href:     path1.Href,
		},
		AuthorizationContext: &pb.AuthorizationContext{},
		CommandMetadata:      &pb.CommandMetadata{},
	}

	commandUnpub3 := pb.UnpublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: path3.DeviceID,
			Href:     path3.Href,
		},
		AuthorizationContext: &pb.AuthorizationContext{},
		CommandMetadata:      &pb.CommandMetadata{},
	}

	/*
		path2topics := func(path Path, event event.Event) []string {
			return topics
		}
	*/

	a1, err := aggregate.NewAggregate(path1.DeviceID, utils.MakeResourceId(path1.DeviceID, path1.Href), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{ResourceStateSnapshotTaken: pb.ResourceStateSnapshotTaken{Id: utils.MakeResourceId(path1.DeviceID, path1.Href), Resource: &pb.Resource{
			DeviceId: path1.DeviceID,
		}, EventMetadata: &pb.EventMetadata{}}}, nil
	}, nil)
	require.NoError(t, err)

	evs, err := a1.HandleCommand(ctx, &commandPub1)
	require.NoError(t, err)
	require.NotNil(t, evs)

	snapshotEventType := func() string {
		s := &events.ResourceStateSnapshotTaken{}
		return s.SnapshotEventType()
	}

	a2, err := aggregate.NewAggregate(path2.DeviceID, utils.MakeResourceId(path2.DeviceID, path2.Href), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{ResourceStateSnapshotTaken: pb.ResourceStateSnapshotTaken{Id: utils.MakeResourceId(path2.DeviceID, path2.Href), Resource: &pb.Resource{
			DeviceId: path2.DeviceID,
		}, EventMetadata: &pb.EventMetadata{}}}, nil
	}, nil)
	require.NoError(t, err)

	evs, err = a2.HandleCommand(ctx, &commandPub2)
	require.NoError(t, err)
	require.NotNil(t, evs)

	projection, err := newProjection(ctx, store, "testProjection", subscriber, func(context.Context) (eventstore.Model, error) { return &mockEventHandler{}, nil }, nil)
	require.NoError(t, err)

	err = projection.Project(ctx, []eventstore.SnapshotQuery{
		eventstore.SnapshotQuery{
			GroupID:           path1.DeviceID,
			AggregateID:       utils.MakeResourceId(path1.DeviceID, path1.Href),
			SnapshotEventType: snapshotEventType(),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(projection.Models(nil)))

	err = projection.Project(ctx, []eventstore.SnapshotQuery{
		eventstore.SnapshotQuery{
			GroupID:           path2.DeviceID,
			AggregateID:       utils.MakeResourceId(path2.DeviceID, path2.Href),
			SnapshotEventType: snapshotEventType(),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(projection.Models(nil)))

	err = projection.SubscribeTo(topics)
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	a3, err := aggregate.NewAggregate(path3.DeviceID, utils.MakeResourceId(path3.DeviceID, path3.Href), aggregate.NewDefaultRetryFunc(1), numEventsInSnapshot, store, func(context.Context) (aggregate.AggregateModel, error) {
		return &events.ResourceStateSnapshotTaken{ResourceStateSnapshotTaken: pb.ResourceStateSnapshotTaken{Id: utils.MakeResourceId(path3.DeviceID, path3.Href), Resource: &pb.Resource{
			DeviceId: path3.DeviceID,
		}, EventMetadata: &pb.EventMetadata{}}}, nil
	}, nil)
	require.NoError(t, err)

	evs, err = a3.HandleCommand(ctx, &commandPub3)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, path3.DeviceID, utils.MakeResourceId(path3.DeviceID, path3.Href), e)
		require.NoError(t, err)
	}
	time.Sleep(time.Second)

	require.Equal(t, 3, len(projection.Models(nil)))

	evs, err = a1.HandleCommand(ctx, &commandUnpub1)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics, path3.DeviceID, utils.MakeResourceId(path3.DeviceID, path3.Href), e)
		require.NoError(t, err)
	}

	err = projection.SubscribeTo(topics[0:1])
	require.NoError(t, err)

	time.Sleep(waitForSubscription)

	err = projection.Forget([]eventstore.SnapshotQuery{
		eventstore.SnapshotQuery{
			GroupID:           path3.DeviceID,
			AggregateID:       utils.MakeResourceId(path3.DeviceID, path3.Href),
			SnapshotEventType: snapshotEventType(),
		},
	})
	require.NoError(t, err)

	evs, err = a3.HandleCommand(ctx, &commandUnpub3)
	require.NoError(t, err)
	require.NotNil(t, evs)
	for _, e := range evs {
		err = publisher.Publish(ctx, topics[1:], path3.DeviceID, utils.MakeResourceId(path3.DeviceID, path3.Href), e)
		require.NoError(t, err)
	}

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
		err = publisher.Publish(ctx, topics, path1.DeviceID, utils.MakeResourceId(path1.DeviceID, path1.Href), e)
		require.NoError(t, err)
	}

	projection.lock.Lock()
	require.Equal(t, 2, len(projection.Models(nil)))
	projection.lock.Unlock()
}

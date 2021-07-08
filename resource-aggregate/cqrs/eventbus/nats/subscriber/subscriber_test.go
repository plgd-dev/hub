package subscriber_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestSubscriber(t *testing.T) {
	publishTopics := []string{"test.subscriber.topic0." + uuid.Must(uuid.NewV4()).String(), "test.subscriber.topic1." + uuid.Must(uuid.NewV4()).String()}
	subscriberTopics := []string{"test.subscriber.topic0.>", "test.subscriber.*.>"}

	timeout := time.Second * 30

	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)

	publisher, err := publisher.New(config.MakePublisherConfig(), logger, publisher.WithMarshaler(json.Marshal))
	require.NoError(t, err)
	require.NotNil(t, publisher)
	defer publisher.Close()

	require.NoError(t, err)
	require.NotNil(t, publisher)
	defer publisher.Close()

	subscriber, err := subscriber.New(config.MakeSubscriberConfig(), logger, subscriber.WithGoPool(func(f func()) error { go f(); return nil }), subscriber.WithUnmarshaler(json.Unmarshal))
	require.NotNil(t, subscriber)
	require.NoError(t, err)
	defer subscriber.Close()

	acceptanceTest(t, context.Background(), timeout, publishTopics, subscriberTopics, publisher, subscriber)
}

type mockEvent struct {
	VersionI     uint64
	EventTypeI   string
	AggregateIDI string
	groupID      string
	isSnapshot   bool
	timestamp    int64
	Data         string
}

func (e mockEvent) Version() uint64 {
	return e.VersionI
}

func (e mockEvent) EventType() string {
	return e.EventTypeI
}

func (e mockEvent) AggregateID() string {
	return e.AggregateIDI
}
func (e mockEvent) GroupID() string {
	return e.groupID
}

func (e mockEvent) IsSnapshot() bool {
	return e.isSnapshot
}

func (e mockEvent) Timestamp() time.Time {
	return time.Unix(0, e.timestamp)
}

type mockEventHandler struct {
	newEvent chan mockEvent
}

func newMockEventHandler() *mockEventHandler {
	return &mockEventHandler{newEvent: make(chan mockEvent, 10)}
}

func (eh *mockEventHandler) Handle(ctx context.Context, iter eventbus.Iter) error {
	for {
		eu, ok := iter.Next(ctx)
		if !ok {
			break
		}

		if eu.EventType() == "" {
			return errors.New("cannot determine type of event")
		}
		var e mockEvent
		err := eu.Unmarshal(&e)
		if err != nil {
			return err
		}
		eh.newEvent <- e
	}

	return iter.Err()
}

func (eh *mockEventHandler) waitForEvent(timeout time.Duration) (mockEvent, error) {
	select {
	case e := <-eh.newEvent:
		return e, nil
	case <-time.After(timeout):
		return mockEvent{}, fmt.Errorf("timeout")
	}
}

func testWaitForAnyEvent(timeout time.Duration, eh1 *mockEventHandler, eh2 *mockEventHandler) (mockEvent, error) {
	select {
	case e := <-eh1.newEvent:
		return e, nil
	case e := <-eh2.newEvent:
		return e, nil
	case <-time.After(timeout):
		return mockEvent{}, fmt.Errorf("timeout")
	}
}

func testNewSubscription(t *testing.T, ctx context.Context, subscriber eventbus.Subscriber, subscriptionId string, publishTopics []string) (*mockEventHandler, eventbus.Observer) {
	t.Log("Subscribe to testNewSubscription")
	m := newMockEventHandler()
	ob, err := subscriber.Subscribe(ctx, subscriptionId, publishTopics, m)
	require.NoError(t, err)
	require.NotNil(t, ob)
	if ob == nil {
		return nil, nil
	}
	return m, ob
}

// AcceptanceTest is the acceptance test that all implementations of publisher, subscriber
// should pass. It should manually be called from a test case in each
// implementation:
//
//   func TestSubscriber(t *testing.T) {
//       ctx := context.Background() // Or other when testing namespaces.
//       publisher := NewPublisher()
//       subscriber := NewSubscriber()
//       timeout := time.Second*5
//       publishTopics := []string{"a", "b"}
//       eventbus.AcceptanceTest(t, ctx, timeout, publishTopics, publisher, subscriber)
//   }
//

type Path struct {
	AggregateId string
	GroupId     string
}

func acceptanceTest(t *testing.T, ctx context.Context, timeout time.Duration, publishTopics, subscribeTopics []string, publisher eventbus.Publisher, subscriber eventbus.Subscriber) {
	//savedEvents := []Event{}
	AggregateID1 := "aggregateID1"
	AggregateID2 := "aggregateID2"

	aggregateID1Path := Path{
		AggregateId: AggregateID1,
		GroupId:     "deviceId",
	}
	/*
		aggregateID2Path := protoEvent.Path{
			AggregateId: AggregateID2,
			Path:        []string{"deviceId"},
		}
		aggregateIDNotExistPath := protoEvent.Path{
			AggregateId: "notExist",
			Path:        []string{"deviceId"},
		}
	*/
	eventsToPublish := []mockEvent{
		{
			EventTypeI:   "test0",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     1,
			EventTypeI:   "test1",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     2,
			EventTypeI:   "test2",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     3,
			EventTypeI:   "test3",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     4,
			EventTypeI:   "test4",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     5,
			EventTypeI:   "test5",
			AggregateIDI: AggregateID1,
		},
		{
			VersionI:     6,
			EventTypeI:   "test6",
			AggregateIDI: AggregateID2,
		},
	}

	require.Equal(t, 2, len(publishTopics))

	t.Log("Without subscription")
	err := publisher.Publish(ctx, publishTopics[0:1], aggregateID1Path.GroupId, aggregateID1Path.AggregateId, eventsToPublish[0])
	require.NoError(t, err)

	// Add handlers and observers.
	t.Log("Subscribe to first topic")
	m0, ob0 := testNewSubscription(t, ctx, subscriber, "sub-0", subscribeTopics[0:1])

	err = publisher.Publish(ctx, publishTopics[0:1], aggregateID1Path.GroupId, aggregateID1Path.AggregateId, eventsToPublish[1])
	require.NoError(t, err)

	event0, err := m0.waitForEvent(timeout)
	require.NoError(t, err)
	require.Equal(t, eventsToPublish[1], event0)

	err = ob0.Close()
	require.NoError(t, err)
	t.Log("Subscribe more observers")
	m1, ob1 := testNewSubscription(t, ctx, subscriber, "sub-1", subscribeTopics[1:2])
	defer func() {
		err = ob1.Close()
		require.NoError(t, err)
	}()
	m2, ob2 := testNewSubscription(t, ctx, subscriber, "sub-2", subscribeTopics[1:2])
	defer func() {
		err = ob2.Close()
		require.NoError(t, err)
	}()
	m3, ob3 := testNewSubscription(t, ctx, subscriber, "sub-shared", subscribeTopics[0:1])
	defer func() {
		err = ob3.Close()
		require.NoError(t, err)
	}()
	m4, ob4 := testNewSubscription(t, ctx, subscriber, "sub-shared", subscribeTopics[0:1])
	defer func() {
		err = ob4.Close()
		require.NoError(t, err)
	}()

	err = publisher.Publish(ctx, publishTopics, aggregateID1Path.GroupId, aggregateID1Path.AggregateId, eventsToPublish[2])
	require.NoError(t, err)

	event1, err := m1.waitForEvent(timeout)
	require.NoError(t, err)
	require.Equal(t, eventsToPublish[2], event1)

	event2, err := m2.waitForEvent(timeout)
	require.NoError(t, err)
	require.Equal(t, eventsToPublish[2], event2)

	event3, err := testWaitForAnyEvent(timeout, m3, m4)
	require.NoError(t, err)
	require.Equal(t, eventsToPublish[2], event3)

	topic := "new_topic_" + uuid.Must(uuid.NewV4()).String()
	publishTopics = append(publishTopics, topic)
	err = ob4.SetTopics(ctx, publishTopics)
	require.NoError(t, err)

	err = publisher.Publish(ctx, []string{topic}, aggregateID1Path.GroupId, aggregateID1Path.AggregateId, eventsToPublish[3])
	require.NoError(t, err)

	event4, err := m4.waitForEvent(timeout)
	require.NoError(t, err)
	require.Equal(t, eventsToPublish[3], event4)

	err = ob4.SetTopics(ctx, nil)
	require.NoError(t, err)
}

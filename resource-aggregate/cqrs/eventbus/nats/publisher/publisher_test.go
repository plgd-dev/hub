package publisher_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher(t *testing.T) {
	topics := []string{"test.subscriber_topic0" + uuid.Must(uuid.NewRandom()).String(), "test.subscriber_topic1" + uuid.Must(uuid.NewRandom()).String()}

	timeout := time.Second * 30
	waitForSubscription := time.Millisecond * 100

	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	naPubClient, publisher, err := test.NewClientAndPublisher(client.ConfigPublisher{
		Config: client.Config{
			URL:            "nats://localhost:4222",
			TLS:            config.MakeTLSClientConfig(),
			FlusherTimeout: time.Second * 30,
		},
	}, fileWatcher, logger, publisher.WithMarshaler(json.Marshal))
	assert.NoError(t, err)
	assert.NotNil(t, publisher)
	defer func() {
		publisher.Close()
		naPubClient.Close()
	}()

	naSubClient, subscriber, err := test.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher,
		logger,
		subscriber.WithGoPool(func(f func()) error { go f(); return nil }),
		subscriber.WithUnmarshaler(json.Unmarshal))
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)
	defer func() {
		subscriber.Close()
		naSubClient.Close()
	}()

	acceptanceTest(context.Background(), t, timeout, waitForSubscription, topics, publisher, subscriber)
}

func TestPublisherJetStream(t *testing.T) {
	topics := []string{"test.subscriber_topic0" + uuid.Must(uuid.NewRandom()).String(), "test.subscriber_topic1" + uuid.Must(uuid.NewRandom()).String()}

	timeout := time.Second * 30
	waitForSubscription := time.Millisecond * 100

	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	cfg := config.MakeTLSClientConfig()
	caPoolArray, err := cfg.CAPoolArray()
	require.NoError(t, err)
	conn, err := nats.Connect("nats://localhost:4222", nats.ClientCert(cfg.CertFile, cfg.KeyFile), nats.RootCAs(caPoolArray...))
	require.NoError(t, err)
	defer conn.Close()
	js, err := conn.JetStream()
	require.NoError(t, err)
	s, err := js.AddStream(&nats.StreamConfig{
		Name:     "TEST",
		Subjects: []string{"test.>"},
	})
	require.NoError(t, err)
	defer func() {
		_ = js.DeleteStream(s.Config.Name)
	}()
	naPubClient, publisher, err := test.NewClientAndPublisher(client.ConfigPublisher{
		Config: client.Config{
			URL:            "nats://localhost:4222",
			TLS:            config.MakeTLSClientConfig(),
			FlusherTimeout: time.Second * 30,
		},
		JetStream: true,
	}, fileWatcher, logger, publisher.WithMarshaler(json.Marshal))
	require.NoError(t, err)
	assert.NotNil(t, publisher)
	defer func() {
		publisher.Close()
		naPubClient.Close()
	}()

	naSubClient, subscriber, err := test.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher,
		logger,
		subscriber.WithGoPool(func(f func()) error { go f(); return nil }),
		subscriber.WithUnmarshaler(json.Unmarshal))
	assert.NoError(t, err)
	assert.NotNil(t, subscriber)
	defer func() {
		subscriber.Close()
		naSubClient.Close()
	}()

	acceptanceTest(context.Background(), t, timeout, waitForSubscription, topics, publisher, subscriber)
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

func testNewSubscription(ctx context.Context, t *testing.T, subscriber eventbus.Subscriber, subscriptionID string, topics []string) (*mockEventHandler, eventbus.Observer) {
	t.Log("Subscribe to testNewSubscription")
	m := newMockEventHandler()
	ob, err := subscriber.Subscribe(ctx, subscriptionID, topics, m)
	assert.NoError(t, err)
	assert.NotNil(t, ob)
	if ob == nil {
		return nil, nil
	}
	return m, ob
}

// AcceptanceTest is the acceptance test that all implementations of publisher, subscriber
// should pass. It should manually be called from a test case in each
// implementation:
//
//	func TestSubscriber(t *testing.T) {
//	    ctx := context.Background() // Or other when testing namespaces.
//	    publisher := NewPublisher()
//	    subscriber := NewSubscriber()
//	    timeout := time.Second*5
//	    topics := []string{"a", "b"}
//	    acceptanceTest(ctx, t, timeout, topics, publisher, subscriber)
//	}
func acceptanceTest(ctx context.Context, t *testing.T, timeout time.Duration, waitForSubscription time.Duration, topics []string, publisher eventbus.Publisher, subscriber eventbus.Subscriber) {
	// savedEvents := []Event{}
	AggregateID1 := "aggregateID1"
	AggregateID2 := "aggregateID2"
	type Path struct {
		AggregateID string
		GroupID     string
	}

	aggregateID1Path := Path{
		AggregateID: AggregateID1,
		GroupID:     "deviceId",
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

	assert.Equal(t, 2, len(topics))

	t.Log("Without subscription")
	err := publisher.Publish(ctx, topics[0:1], aggregateID1Path.GroupID, aggregateID1Path.AggregateID, eventsToPublish[0])
	assert.NoError(t, err)
	time.Sleep(waitForSubscription)

	// Add handlers and observers.
	t.Log("Subscribe to first topic")
	m0, ob0 := testNewSubscription(ctx, t, subscriber, "sub-0", topics[0:1])
	time.Sleep(waitForSubscription)

	err = publisher.Publish(ctx, topics[0:1], aggregateID1Path.GroupID, aggregateID1Path.AggregateID, eventsToPublish[1])
	assert.NoError(t, err)

	event0, err := m0.waitForEvent(timeout)
	assert.NoError(t, err)
	assert.Equal(t, eventsToPublish[1], event0)

	err = ob0.Close()
	assert.NoError(t, err)
	t.Log("Subscribe more observers")
	m1, ob1 := testNewSubscription(ctx, t, subscriber, "sub-1", topics[1:2])
	defer func() {
		err = ob1.Close()
		assert.NoError(t, err)
	}()
	m2, ob2 := testNewSubscription(ctx, t, subscriber, "sub-2", topics[1:2])
	defer func() {
		err = ob2.Close()
		assert.NoError(t, err)
	}()
	m3, ob3 := testNewSubscription(ctx, t, subscriber, "sub-shared", topics[0:1])
	defer func() {
		err = ob3.Close()
		assert.NoError(t, err)
	}()
	m4, ob4 := testNewSubscription(ctx, t, subscriber, "sub-shared", topics[0:1])
	defer func() {
		err = ob4.Close()
		assert.NoError(t, err)
	}()

	time.Sleep(waitForSubscription)

	err = publisher.Publish(ctx, topics, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, eventsToPublish[2])
	assert.NoError(t, err)

	event1, err := m1.waitForEvent(timeout)
	assert.NoError(t, err)
	assert.Equal(t, eventsToPublish[2], event1)

	event2, err := m2.waitForEvent(timeout)
	assert.NoError(t, err)
	assert.Equal(t, eventsToPublish[2], event2)

	event3, err := testWaitForAnyEvent(timeout, m3, m4)
	assert.NoError(t, err)
	assert.Equal(t, eventsToPublish[2], event3)

	topic := "test.new_topic_" + uuid.Must(uuid.NewRandom()).String()
	topics = append(topics, topic)
	err = ob4.SetTopics(ctx, topics)
	time.Sleep(waitForSubscription)
	assert.NoError(t, err)

	err = publisher.Publish(ctx, []string{topic}, aggregateID1Path.GroupID, aggregateID1Path.AggregateID, eventsToPublish[3])
	assert.NoError(t, err)

	event4, err := m4.waitForEvent(timeout)
	assert.NoError(t, err)
	assert.Equal(t, eventsToPublish[3], event4)

	err = ob4.SetTopics(ctx, nil)
	assert.NoError(t, err)
}

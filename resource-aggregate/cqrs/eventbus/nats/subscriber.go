package nats

import (
	"context"
	"fmt"
	"sync"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/protobuf/proto"
)

//UnmarshalerFunc unmarshal bytes to pointer of struct.
type UnmarshalerFunc = func(b []byte, v interface{}) error

// Subscriber implements a eventbus.Subscriber interface.
type Subscriber struct {
	clientId        string
	dataUnmarshaler UnmarshalerFunc
	errFunc         eventbus.ErrFunc
	conn            *nats.Conn
	url             string
	goroutinePoolGo eventbus.GoroutinePoolGoFunc
}

//Observer handles events from kafka
type Observer struct {
	lock            sync.Mutex
	dataUnmarshaler UnmarshalerFunc
	eventHandler    eventbus.Handler
	errFunc         eventbus.ErrFunc
	conn            *nats.Conn
	subscriptionId  string
	subs            map[string]*nats.Subscription
}

// NewSubscriber create new subscriber with proto unmarshaller.
func NewSubscriber(config Config, goroutinePoolGo eventbus.GoroutinePoolGoFunc, errFunc eventbus.ErrFunc, opts ...Option) (*Subscriber, error) {
	for _, o := range opts {
		config = o(config)
	}

	s, err := newSubscriber(config.URL, utils.Unmarshal, goroutinePoolGo, errFunc, config.Options...)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// NewSubscriber creates a subscriber.
func newSubscriber(url string, eventUnmarshaler UnmarshalerFunc, goroutinePoolGo eventbus.GoroutinePoolGoFunc, errFunc eventbus.ErrFunc, options ...nats.Option) (*Subscriber, error) {
	if eventUnmarshaler == nil {
		return nil, fmt.Errorf("invalid eventUnmarshaler")
	}
	if errFunc == nil {
		return nil, fmt.Errorf("invalid errFunc")
	}

	conn, err := nats.Connect(url, options...)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %w", err)
	}

	return &Subscriber{
		dataUnmarshaler: eventUnmarshaler,
		errFunc:         errFunc,
		conn:            conn,
		goroutinePoolGo: goroutinePoolGo,
	}, nil
}

// Subscribe creates a observer that listen on events from topics.
func (b *Subscriber) Subscribe(ctx context.Context, subscriptionId string, topics []string, eh eventbus.Handler) (eventbus.Observer, error) {
	observer := b.newObservation(ctx, subscriptionId, eventbus.NewGoroutinePoolHandler(b.goroutinePoolGo, eh, b.errFunc))

	err := observer.SetTopics(ctx, topics)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe: %w", err)
	}

	return observer, nil
}

// Close closes subscriber.
func (b *Subscriber) Close() {
	b.conn.Close()
}

func (b *Subscriber) newObservation(ctx context.Context, subscriptionId string, eh eventbus.Handler) *Observer {
	return &Observer{
		conn:            b.conn,
		dataUnmarshaler: b.dataUnmarshaler,
		subscriptionId:  subscriptionId,
		subs:            make(map[string]*nats.Subscription),
		eventHandler:    eh,
		errFunc:         b.errFunc,
	}
}

func (o *Observer) cleanUp(topics map[string]bool) (map[string]bool, error) {
	var errors []error
	for topic, sub := range o.subs {
		if _, ok := topics[topic]; !ok {
			err := sub.Unsubscribe()
			if err != nil {
				errors = append(errors, err)
			}
			delete(o.subs, topic)
		}
	}
	newSubs := make(map[string]bool)
	for topic := range topics {
		if _, ok := o.subs[topic]; !ok {
			newSubs[topic] = true
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("cannot unsubscribe from topics: %v", errors)
	}
	return newSubs, nil
}

// SetTopics set new topics to observe.
func (o *Observer) SetTopics(ctx context.Context, topics []string) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	mapTopics := make(map[string]bool)
	for _, topic := range topics {
		mapTopics[topic] = true
	}

	newTopicsForSub, err := o.cleanUp(mapTopics)
	if err != nil {
		return fmt.Errorf("cannot set topics: %w", err)
	}
	for topic := range newTopicsForSub {
		sub, err := o.conn.QueueSubscribe(topic, o.subscriptionId, o.handleMsg)
		if err != nil {
			o.cleanUp(make(map[string]bool))
			return fmt.Errorf("cannot subscribe to topics: %w", err)
		}
		o.subs[topic] = sub
	}

	return nil
}

// Close cancel observation and close connection to kafka.
func (o *Observer) Close() error {
	o.lock.Lock()
	defer o.lock.Unlock()
	_, err := o.cleanUp(make(map[string]bool))
	if err != nil {
		return fmt.Errorf("cannot close observer: %w", err)
	}
	return nil
}

func (o *Observer) handleMsg(msg *nats.Msg) {
	var e pb.Event

	err := proto.Unmarshal(msg.Data, &e)
	if err != nil {
		o.errFunc(fmt.Errorf("cannot unmarshal event: %w", err))
		return
	}

	i := iter{
		hasNext: true,
		e:       e,
		dataUnmarshaler: func(v interface{}) error {
			return o.dataUnmarshaler(e.Data, v)
		},
	}

	if err := o.eventHandler.Handle(context.Background(), &i); err != nil {
		o.errFunc(fmt.Errorf("cannot unmarshal event: %w", err))
	}
}

type eventUnmarshaler struct {
	version         uint64
	eventType       string
	aggregateID     string
	groupID         string
	dataUnmarshaler func(v interface{}) error
}

func (e *eventUnmarshaler) Version() uint64 {
	return e.version
}
func (e *eventUnmarshaler) EventType() string {
	return e.eventType
}
func (e *eventUnmarshaler) AggregateID() string {
	return e.aggregateID
}
func (e *eventUnmarshaler) GroupID() string {
	return e.groupID
}
func (e *eventUnmarshaler) Unmarshal(v interface{}) error {
	return e.dataUnmarshaler(v)
}

type iter struct {
	e               pb.Event
	dataUnmarshaler func(v interface{}) error
	hasNext         bool
}

func (i *iter) Next(ctx context.Context) (eventbus.EventUnmarshaler, bool) {
	if i.hasNext {
		i.hasNext = false
		return &eventUnmarshaler{
			version:         i.e.Version,
			aggregateID:     i.e.AggregateId,
			eventType:       i.e.EventType,
			groupID:         i.e.GroupId,
			dataUnmarshaler: i.dataUnmarshaler,
		}, true
	}
	return nil, false
}

func (i *iter) Err() error {
	return nil
}

package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/pb"
	"google.golang.org/protobuf/proto"
)

// UnmarshalerFunc unmarshal bytes to pointer of struct.
type UnmarshalerFunc = func(s []byte, v interface{}) error

// ReconnectFunc called when reconnect occurs
type ReconnectFunc func()

type reconnect struct {
	id uint64
	f  ReconnectFunc
}

// Subscriber implements a eventbus.Subscriber interface.
type Subscriber struct {
	dataUnmarshaler UnmarshalerFunc
	logger          log.Logger
	conn            *nats.Conn
	goroutinePoolGo eventbus.GoroutinePoolGoFunc
	closeFunc       []func()
	pendingLimits   natsClient.PendingLimitsConfig

	lock        sync.Mutex
	reconnectID uint64
	reconnect   []reconnect
}

func (s *Subscriber) AddCloseFunc(f func()) {
	s.closeFunc = append(s.closeFunc, f)
}

func (s *Subscriber) AddReconnectFunc(f func()) uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.reconnectID++
	s.reconnect = append(s.reconnect, reconnect{
		id: s.reconnectID,
		f:  f,
	})
	return s.reconnectID
}

func (s *Subscriber) RemoveReconnectFunc(id uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for idx := range s.reconnect {
		if s.reconnect[idx].id == id {
			s.reconnect = append(s.reconnect[:idx], s.reconnect[idx+1:]...)
			break
		}
	}
}

func (s *Subscriber) reconnectCopy() []reconnect {
	s.lock.Lock()
	defer s.lock.Unlock()
	reconnect := make([]reconnect, len(s.reconnect))
	copy(reconnect, s.reconnect)
	return reconnect
}

func (s *Subscriber) reconnectedHandler(c *nats.Conn) {
	reconnect := s.reconnectCopy()
	for idx := range reconnect {
		reconnect[idx].f()
	}
}

// Observer handles events from nats
type Observer struct {
	lock            sync.Mutex
	dataUnmarshaler UnmarshalerFunc
	eventHandler    eventbus.Handler
	logger          log.Logger
	conn            *nats.Conn
	subscriptionID  string
	subs            map[string]*nats.Subscription
	ctx             context.Context
	cancel          context.CancelFunc
	pendingLimits   natsClient.PendingLimitsConfig
}

type options struct {
	dataUnmarshaler UnmarshalerFunc
	goroutinePoolGo eventbus.GoroutinePoolGoFunc
}

type Option interface {
	apply(o *options)
}

type UnmarshalerOpt struct {
	dataUnmarshaler UnmarshalerFunc
}

func (o UnmarshalerOpt) apply(opts *options) {
	opts.dataUnmarshaler = o.dataUnmarshaler
}

func WithUnmarshaler(dataUnmarshaler UnmarshalerFunc) UnmarshalerOpt {
	return UnmarshalerOpt{
		dataUnmarshaler: dataUnmarshaler,
	}
}

type GoroutinePoolGoOpt struct {
	goroutinePoolGo eventbus.GoroutinePoolGoFunc
}

func (o GoroutinePoolGoOpt) apply(opts *options) {
	opts.goroutinePoolGo = o.goroutinePoolGo
}

func WithGoPool(goroutinePoolGo eventbus.GoroutinePoolGoFunc) GoroutinePoolGoOpt {
	return GoroutinePoolGoOpt{
		goroutinePoolGo: goroutinePoolGo,
	}
}

// Create publisher with existing NATS connection and proto marshaller
func New(conn *nats.Conn, pendingLimits natsClient.PendingLimitsConfig, logger log.Logger, opts ...Option) (*Subscriber, error) {
	cfg := options{
		dataUnmarshaler: json.Unmarshal,
		goroutinePoolGo: nil,
	}
	for _, o := range opts {
		o.apply(&cfg)
	}

	if cfg.dataUnmarshaler == nil {
		return nil, fmt.Errorf("invalid eventUnmarshaler")
	}

	s := &Subscriber{
		dataUnmarshaler: cfg.dataUnmarshaler,
		logger:          logger,
		conn:            conn,
		goroutinePoolGo: cfg.goroutinePoolGo,
		pendingLimits:   pendingLimits,
		reconnect:       make([]reconnect, 0, 8),
	}
	conn.SetReconnectHandler(s.reconnectedHandler)

	return s, nil
}

// Subscribe creates a observer that listen on events from topics.
func (s *Subscriber) Subscribe(ctx context.Context, subscriptionID string, topics []string, eh eventbus.Handler) (eventbus.Observer, error) {
	observer := s.newObservation(subscriptionID, eventbus.NewGoroutinePoolHandler(s.goroutinePoolGo, eh, func(err error) { s.logger.Error(err) }))

	err := observer.SetTopics(ctx, topics)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe: %w", err)
	}

	return observer, nil
}

func (s *Subscriber) Close() {
	for _, f := range s.closeFunc {
		f()
	}
}

func (s *Subscriber) newObservation(subscriptionID string, eh eventbus.Handler) *Observer {
	ctx, cancel := context.WithCancel(context.Background())
	o := &Observer{
		conn:            s.conn,
		dataUnmarshaler: s.dataUnmarshaler,
		subscriptionID:  subscriptionID,
		subs:            make(map[string]*nats.Subscription),
		eventHandler:    eh,
		logger:          s.logger,
		ctx:             ctx,
		cancel:          cancel,
		pendingLimits:   s.pendingLimits,
	}

	return o
}

func (o *Observer) cleanUp(topics map[string]bool) (map[string]bool, error) {
	var errors []error
	var unsetTopics bool
	for topic, sub := range o.subs {
		if _, ok := topics[topic]; !ok {
			err := sub.Unsubscribe()
			if err != nil {
				errors = append(errors, err)
			}
			delete(o.subs, topic)
			unsetTopics = true
		}
	}
	if unsetTopics {
		if err := o.conn.Flush(); err != nil {
			errors = append(errors, err)
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

	cleanUpAfterError := func(err error) error {
		errors := []error{
			fmt.Errorf("cannot subscribe to topics: %w", err),
		}
		if _, err := o.cleanUp(make(map[string]bool)); err != nil {
			errors = append(errors, err)
		}
		return fmt.Errorf("%+v", errors)
	}

	for topic := range newTopicsForSub {
		sub, err := o.conn.QueueSubscribe(topic, o.subscriptionID, o.handleMsg)
		if err != nil {
			return cleanUpAfterError(err)
		}
		o.subs[topic] = sub
		err = sub.SetPendingLimits(o.pendingLimits.MsgLimit, o.pendingLimits.BytesLimit)
		if err != nil {
			return cleanUpAfterError(err)
		}
	}

	if len(newTopicsForSub) > 0 {
		return o.conn.Flush()
	}
	return nil
}

// Close cancel observation and close connection to nats.
func (o *Observer) Close() error {
	o.cancel()
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
		o.logger.Errorf("cannot unmarshal event: %v", err)
		return
	}

	i := iter{
		hasNext: true,
		e:       &e,
		dataUnmarshaler: func(v interface{}) error {
			return o.dataUnmarshaler(e.Data, v)
		},
	}

	if err := o.eventHandler.Handle(context.Background(), &i); err != nil {
		o.logger.Errorf("cannot unmarshal event: %v", err)
	}
}

type eventUnmarshaler struct {
	pb              *pb.Event
	dataUnmarshaler func(v interface{}) error
}

func (e *eventUnmarshaler) Version() uint64 {
	return e.pb.GetVersion()
}

func (e *eventUnmarshaler) EventType() string {
	return e.pb.GetEventType()
}

func (e *eventUnmarshaler) AggregateID() string {
	return e.pb.GetAggregateId()
}

func (e *eventUnmarshaler) GroupID() string {
	return e.pb.GetGroupId()
}

func (e *eventUnmarshaler) IsSnapshot() bool {
	return e.pb.GetIsSnapshot()
}

func (e *eventUnmarshaler) Timestamp() time.Time {
	return pkgTime.Unix(0, e.pb.GetTimestamp())
}

func (e *eventUnmarshaler) Unmarshal(v interface{}) error {
	return e.dataUnmarshaler(v)
}

type iter struct {
	e               *pb.Event
	dataUnmarshaler func(v interface{}) error
	hasNext         bool
}

func (i *iter) Next(ctx context.Context) (eventbus.EventUnmarshaler, bool) {
	if i.hasNext {
		i.hasNext = false
		return &eventUnmarshaler{
			pb:              i.e,
			dataUnmarshaler: i.dataUnmarshaler,
		}, true
	}
	return nil, false
}

func (i *iter) Err() error {
	return nil
}

func (s *Subscriber) Conn() *nats.Conn {
	return s.conn
}

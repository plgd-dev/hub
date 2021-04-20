package subscriber

import (
	"context"
	"fmt"
	"sync"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

//UnmarshalerFunc unmarshal bytes to pointer of struct.
type UnmarshalerFunc = func(b []byte, v interface{}) error

// Subscriber implements a eventbus.Subscriber interface.
type Subscriber struct {
	clientId        string
	dataUnmarshaler UnmarshalerFunc
	logger          *zap.Logger
	conn            *nats.Conn
	url             string
	goroutinePoolGo eventbus.GoroutinePoolGoFunc
	closeFunc       []func()
}

func (p *Subscriber) AddCloseFunc(f func()) {
	p.closeFunc = append(p.closeFunc, f)
}

//Observer handles events from nats
type Observer struct {
	lock            sync.Mutex
	dataUnmarshaler UnmarshalerFunc
	eventHandler    eventbus.Handler
	logger          *zap.Logger
	conn            *nats.Conn
	subscriptionId  string
	subs            map[string]*nats.Subscription
	ch              chan *nats.Msg
	ctx             context.Context
	cancel          context.CancelFunc
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

// NewSubscriber create new subscriber with proto unmarshaller.
func New(config Config, logger *zap.Logger, opts ...Option) (*Subscriber, error) {
	cfg := options{
		dataUnmarshaler: utils.Unmarshal,
		goroutinePoolGo: nil,
	}
	for _, o := range opts {
		o.apply(&cfg)
	}
	certManager, err := client.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	config.Options = append(config.Options, nats.Secure(certManager.GetTLSConfig()))
	s, err := newSubscriber(config.URL, cfg.dataUnmarshaler, cfg.goroutinePoolGo, logger, config.Options...)
	if err != nil {
		certManager.Close()
		return nil, err
	}
	s.AddCloseFunc(certManager.Close)
	return s, nil
}

// NewSubscriber creates a subscriber.
func newSubscriber(url string, eventUnmarshaler UnmarshalerFunc, goroutinePoolGo eventbus.GoroutinePoolGoFunc, logger *zap.Logger, options ...nats.Option) (*Subscriber, error) {
	if eventUnmarshaler == nil {
		return nil, fmt.Errorf("invalid eventUnmarshaler")
	}

	conn, err := nats.Connect(url, options...)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %w", err)
	}

	return &Subscriber{
		dataUnmarshaler: eventUnmarshaler,
		logger:          logger,
		conn:            conn,
		goroutinePoolGo: goroutinePoolGo,
	}, nil
}

// Subscribe creates a observer that listen on events from topics.
func (b *Subscriber) Subscribe(ctx context.Context, subscriptionId string, topics []string, eh eventbus.Handler) (eventbus.Observer, error) {
	observer := b.newObservation(ctx, subscriptionId, eventbus.NewGoroutinePoolHandler(b.goroutinePoolGo, eh, func(err error) { b.logger.Sugar().Error(err) }))

	err := observer.SetTopics(ctx, topics)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe: %w", err)
	}

	return observer, nil
}

// Close closes subscriber.
func (b *Subscriber) Close() {
	b.conn.Close()
	for _, f := range b.closeFunc {
		f()
	}
}

func (b *Subscriber) newObservation(ctx context.Context, subscriptionId string, eh eventbus.Handler) *Observer {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *nats.Msg, 32)
	o := &Observer{
		conn:            b.conn,
		dataUnmarshaler: b.dataUnmarshaler,
		subscriptionId:  subscriptionId,
		subs:            make(map[string]*nats.Subscription),
		eventHandler:    eh,
		logger:          b.logger,
		ctx:             ctx,
		cancel:          cancel,
		ch:              ch,
	}
	go func() {
		for {
			select {
			case <-o.ctx.Done():
				return
			case msg := <-ch:
				if msg != nil {
					o.handleMsg(msg)
				}
			}
		}
	}()

	return o
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
		sub, err := o.conn.QueueSubscribeSyncWithChan(topic, o.subscriptionId, o.ch)
		if err != nil {
			o.cleanUp(make(map[string]bool))
			return fmt.Errorf("cannot subscribe to topics: %w", err)
		}
		o.subs[topic] = sub
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
	if o.ch != nil {
		close(o.ch)
		o.ch = nil
	}
	return nil
}

func (o *Observer) handleMsg(msg *nats.Msg) {
	var e pb.Event

	err := proto.Unmarshal(msg.Data, &e)
	if err != nil {
		o.logger.Sugar().Errorf("cannot unmarshal event: %v", err)
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
		o.logger.Sugar().Errorf("cannot unmarshal event: %v", err)
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

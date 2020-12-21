package jetstream

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/plgd-dev/cqrs/eventstore/maintenance"
	uuid "github.com/satori/go.uuid"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
	cqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	"github.com/plgd-dev/cqrs/event"
	"github.com/plgd-dev/cqrs/eventstore"
	"github.com/plgd-dev/cqrs/protobuf/eventbus"
	"github.com/silenceper/pool"
)

// EventStore implements an EventStore for JetStream.
type EventStore struct {
	pool   pool.Pool
	config Config
}

//NewEventStore create a event store from configuration
func NewEventStore(config Config, goroutinePoolGo eventstore.GoroutinePoolGoFunc, opts ...Option) (*EventStore, error) {
	config.marshalerFunc = cqrsUtils.Marshal
	config.unmarshalerFunc = cqrsUtils.Unmarshal
	config.logDebug = func(fmt string, args ...interface{}) {}
	for _, o := range opts {
		config = o(config)
	}

	//factory Specify the method to create the connection
	factory := func() (interface{}, error) {
		return connect(config)
	}

	//close Specify the method to close the connection
	close := func(v interface{}) error {
		v.(*JsmConn).close()
		return nil
	}

	poolConfig := &pool.Config{
		InitialCap:  config.InitialCap,
		MaxIdle:     config.MaxIdle,
		MaxCap:      config.MaxCap,
		Factory:     factory,
		Close:       close,
		IdleTimeout: config.IdleTimeout,
	}

	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		return nil, err
	}

	return &EventStore{
		config: config,
		pool:   p,
	}, nil
}

// Save saves events to a path.
func (s *EventStore) Save(ctx context.Context, groupID, aggregateID string, events []event.Event) (concurrencyException bool, err error) {
	if len(events) != 1 {
		return false, fmt.Errorf("not supported")
	}
	raw, err := s.config.marshalerFunc(events[0])
	if err != nil {
		return false, fmt.Errorf("cannot marshal event data: %w", err)
	}
	storeEvent := eventbus.Event{
		EventType:   events[0].EventType(),
		Version:     events[0].Version(),
		GroupId:     groupID,
		AggregateId: aggregateID,
		Data:        raw,
	}
	data, err := storeEvent.Marshal()
	if err != nil {
		return false, fmt.Errorf("cannot marshal event: %w", err)
	}

	c, err := s.pool.Get()
	if err != nil {
		return false, fmt.Errorf("cannot get connection: %w", err)
	}
	conn := c.(*JsmConn)

	streamName := "devices_" + groupID + "_" + aggregateID
	subject := "devices." + groupID + "." + aggregateID + ".events"
	cfg := api.StreamConfig{
		Name:         streamName,
		Subjects:     []string{subject},
		MaxMsgs:      -1,
		MaxBytes:     -1,
		MaxMsgSize:   -1,
		Duplicates:   0,
		MaxAge:       time.Hour * 24 * 365,
		Storage:      api.FileStorage,
		NoAck:        false,
		Retention:    api.LimitsPolicy,
		Discard:      api.DiscardOld,
		MaxConsumers: -1,
		Replicas:     1,
	}
	_, err = conn.loadOrNewStreamFromDefault(streamName, cfg)
	if err != nil {
		return false, err
	}

	msg, err := conn.requestWithContext(ctx, subject, data /*, nats.PublishExpectsStream(streamName)*/)
	if err != nil {
		if strings.Contains(err.Error(), "evenstore: concurrency exception:") {
			s.pool.Put(c)
			return true, nil
		}
		return false, err
	}
	if msg.Data != nil && strings.Contains(string(msg.Data), "evenstore: concurrency exception:") {
		s.pool.Put(c)
		return true, nil
	}
	if msg.Data != nil && strings.Contains(string(msg.Data), "error:") {
		return false, fmt.Errorf(string(msg.Data))
	}

	s.pool.Put(c)
	return false, err
}

// SaveSnapshot saves snapshots to a path.
func (s *EventStore) SaveSnapshot(ctx context.Context, groupID, aggregateID string, ev event.Event) (concurrencyException bool, err error) {
	return s.Save(ctx, groupID, aggregateID, []event.Event{ev})
}

type iterator struct {
	events          []*nats.Msg
	dataUnmarshaler event.UnmarshalerFunc
	LogDebugfFunc   LogDebugFunc
	err             error
	idx             int
}

func (i *iterator) Next(ctx context.Context, e *event.EventUnmarshaler) bool {
	if i.err != nil {
		return false
	}
	if i.idx >= len(i.events) {
		return false
	}
	msg := i.events[i.idx]
	i.idx++
	if msg.Data == nil {
		return false
	}

	var storeEvent eventbus.Event
	err := storeEvent.Unmarshal(msg.Data)
	if err != nil {
		i.err = err
		return false
	}

	i.LogDebugfFunc("iterator.next: %+v", storeEvent)

	e.Version = storeEvent.GetVersion()
	e.AggregateId = storeEvent.GetAggregateId()
	e.EventType = storeEvent.GetEventType()
	e.GroupId = storeEvent.GetGroupId()
	e.Unmarshal = func(v interface{}) error {
		return i.dataUnmarshaler(storeEvent.Data, v)
	}
	return true
}

func (i *iterator) Err() error {
	return i.err
}

// LoadFromVersion loads aggragate events from a specific version.
func (s *EventStore) LoadFromVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler event.Handler) error {
	// Receive all stored values in order
	var errors []error
	for _, q := range queries {
		err := func() error {
			c, err := s.pool.Get()
			if err != nil {
				return fmt.Errorf("cannot get connection: %w", err)
			}
			conn := c.(*JsmConn)

			streamNames, err := conn.streamNames(&jsm.StreamNamesFilter{
				Subject: "devices.*." + q.AggregateId + ".events",
			})
			if err != nil {
				return err
			}
			if len(streamNames) == 0 {
				s.pool.Put(c)
				return nil
			}
			events := make([]*nats.Msg, 0, 128)
			consumer, err := conn.newConsumer(streamNames[0], jsm.DurableName(uuid.NewV4().String()), jsm.StartAtSequence(q.Version+1))
			if err != nil {
				return err
			}
			pending, err := consumer.PendingMessages()
			if err != nil {
				consumer.Delete()
				return err
			}
			for i := uint64(0); i < pending; i++ {
				msg, err := consumer.NextMsgContext(ctx)
				if err != nil {
					consumer.Delete()
					return err
				}
				err = msg.Ack()
				if err != nil {
					consumer.Delete()
					return err
				}
				events = append(events, msg)
			}
			consumer.Delete()
			s.pool.Put(c)

			i := iterator{
				events:          events,
				dataUnmarshaler: s.config.unmarshalerFunc,
				LogDebugfFunc:   s.config.logDebug,
			}
			return eventHandler.Handle(ctx, &i)
		}()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}

	return nil
}

// LoadUpToVersion loads aggragate events up to a specific version.
func (s *EventStore) LoadUpToVersion(ctx context.Context, queries []eventstore.VersionQuery, eventHandler event.Handler) error {
	return fmt.Errorf("not supported")
}

// LoadFromSnapshot loads events from beginning.
func (s *EventStore) LoadFromSnapshot(ctx context.Context, queries []eventstore.SnapshotQuery, eventHandler event.Handler) error {
	var vq []eventstore.VersionQuery
	if len(queries) == 0 {
		queries = append(queries, eventstore.SnapshotQuery{})
	}
	for _, q := range queries {
		if q.AggregateId != "" {
			vq = append(vq, eventstore.VersionQuery{
				AggregateId: q.AggregateId,
			})
		} else {
			qID := "devices.*.*.events"
			if q.GroupId != "" {
				qID = "devices." + q.GroupId + ".*.events"
			}
			c, err := s.pool.Get()
			if err != nil {
				return fmt.Errorf("cannot get connection: %w", err)
			}
			conn := c.(*JsmConn)
			streamNames, err := conn.streamNames(&jsm.StreamNamesFilter{
				Subject: qID,
			})
			if err != nil {
				return err
			}
			s.pool.Put(c)
			for _, n := range streamNames {
				ids := strings.Split(n, "_")

				vq = append(vq, eventstore.VersionQuery{
					AggregateId: ids[len(ids)-1],
				})
			}
		}
	}
	if len(vq) == 0 {
		return nil
	}

	return s.LoadFromVersion(ctx, vq, eventHandler)
}

// RemoveUpToVersion deletes the aggragates events up to a specific version.
func (s *EventStore) RemoveUpToVersion(ctx context.Context, queries []eventstore.VersionQuery) error {
	return fmt.Errorf("not supported")
}

// Insert stores (or updates) the information about the latest snapshot version per aggregate into the DB
func (s *EventStore) Insert(ctx context.Context, task maintenance.Task) error {
	return nil
}

// Query retrieves the latest snapshot version per aggregate for thw number of aggregates specified by 'limit'
func (s *EventStore) Query(ctx context.Context, limit int, taskHandler maintenance.TaskHandler) error {
	return fmt.Errorf("not supported")
}

// Remove deletes (the latest snapshot version) database record for a given aggregate ID
func (s *EventStore) Remove(ctx context.Context, task maintenance.Task) error {
	return fmt.Errorf("not supported")
}

// Clear clears the event storage.
func (s *EventStore) Clear(ctx context.Context) error {
	c, err := s.pool.Get()
	if err != nil {
		return fmt.Errorf("cannot get connection: %w", err)
	}
	conn := c.(*JsmConn)
	streams, err := conn.streams()
	if err != nil {
		return err
	}
	var errors []error
	for _, stream := range streams {
		err = stream.Delete()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	s.pool.Put(c)
	return nil
}

// Close closes the database session.
func (s *EventStore) Close(ctx context.Context) error {
	s.pool.Release()
	return nil
}

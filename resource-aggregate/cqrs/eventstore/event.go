package eventstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ETagData struct {
	ETag      []byte
	Timestamp int64
}

// Event interface over event created by user.
type Event = interface {
	Version() uint64
	EventType() string
	AggregateID() string
	GroupID() string
	IsSnapshot() bool
	ServiceID() (string, bool)
	Timestamp() time.Time
	ETag() *ETagData
	Types() []string
}

// EventUnmarshaler provides event.
type EventUnmarshaler = interface {
	Version() uint64
	EventType() string
	AggregateID() string
	GroupID() string
	IsSnapshot() bool
	Timestamp() time.Time
	Unmarshal(v interface{}) error
}

// Iter provides iterator over events from eventstore or eventbus.
type Iter = interface {
	Next(ctx context.Context) (EventUnmarshaler, bool)
	Err() error
}

// Handler provides handler for eventstore or eventbus.
type Handler = interface {
	Handle(ctx context.Context, iter Iter) (err error)
}

type LoadedEvent struct {
	version         uint64
	eventType       string
	aggregateID     string
	groupID         string
	isSnapshot      bool
	timestamp       time.Time
	dataUnmarshaler func(v interface{}) error
}

func NewLoadedEvent(
	version uint64,
	eventType string,
	aggregateID string,
	groupID string,
	isSnapshot bool,
	timestamp time.Time,
	dataUnmarshaler func(v interface{}) error,
) LoadedEvent {
	return LoadedEvent{
		version:         version,
		eventType:       eventType,
		aggregateID:     aggregateID,
		groupID:         groupID,
		isSnapshot:      isSnapshot,
		timestamp:       timestamp,
		dataUnmarshaler: dataUnmarshaler,
	}
}

func (e LoadedEvent) Version() uint64 {
	return e.version
}

func (e LoadedEvent) EventType() string {
	return e.eventType
}

func (e LoadedEvent) AggregateID() string {
	return e.aggregateID
}

func (e LoadedEvent) GroupID() string {
	return e.groupID
}

func (e LoadedEvent) Unmarshal(v interface{}) error {
	return e.dataUnmarshaler(v)
}

func (e LoadedEvent) IsSnapshot() bool {
	return e.isSnapshot
}

func (e LoadedEvent) Timestamp() time.Time {
	return e.timestamp
}

// Validate events before saving them in the database
// Prerequisites that must hold:
//  1. AggregateID, GroupID and EventType are not empty
//  2. Version for each event is by 1 greater than the version of the previous event
//  3. Only the first event can be a snapshot
//  4. All events have the same AggregateID and GroupID
//  5. Timestamps are non-zero
//  6. Timestamps are non-decreasing
func ValidateEventsBeforeSave(events []Event) error {
	if len(events) == 0 || events[0] == nil {
		return fmt.Errorf("invalid events('%v')", events)
	}

	firstEvent := events[0]
	if err := validateFirstEvent(firstEvent); err != nil {
		return err
	}

	aggregateID := firstEvent.AggregateID()
	groupID := firstEvent.GroupID()
	version := firstEvent.Version()
	timestamp := firstEvent.Timestamp()

	for idx, event := range events {
		if event == nil {
			return fmt.Errorf("invalid events[%v]('%v')", idx, event)
		}
		if err := validateEvent(event, idx, aggregateID, groupID, version, timestamp); err != nil {
			return err
		}
		timestamp = event.Timestamp()
	}

	return nil
}

func validateFirstEvent(event Event) error {
	_, err := uuid.Parse(event.AggregateID())
	if err != nil {
		return fmt.Errorf("invalid events[0].AggregateID('%v'): %w", event.AggregateID(), err)
	}

	_, err = uuid.Parse(event.GroupID())
	if err != nil {
		return fmt.Errorf("invalid events[0].GroupID('%v'): %w", event.GroupID(), err)
	}

	if event.Timestamp().IsZero() {
		return errors.New("invalid zero events[0].Timestamp")
	}

	return nil
}

func validateEvent(event Event, idx int, aggregateID, groupID string, version uint64, timestamp time.Time) error {
	if event.AggregateID() != aggregateID {
		return fmt.Errorf("invalid events[%v].AggregateID('%v') != events[0].AggregateID('%v')", idx, event.AggregateID(), aggregateID)
	}

	if event.GroupID() != groupID {
		return fmt.Errorf("invalid events[%v].GroupID('%v') != events[0].GroupID('%v')", idx, event.GroupID(), groupID)
	}

	if event.EventType() == "" {
		return fmt.Errorf("invalid events[%v].EventType('%v')", idx, event.EventType())
	}

	if event.Timestamp().IsZero() {
		return fmt.Errorf("invalid zero events[%v].Timestamp", idx)
	}

	if idx > 0 {
		if event.Version() != version+uint64(idx) {
			return fmt.Errorf("invalid continues ascending events[%v].Version(%v))", idx, event.Version())
		}

		if timestamp.After(event.Timestamp()) {
			return fmt.Errorf("invalid decreasing events[%v].Timestamp(%v))", idx, event.Timestamp())
		}
	}

	if serviceID, ok := event.ServiceID(); ok && serviceID != "" {
		_, err := uuid.Parse(serviceID)
		if err != nil {
			return fmt.Errorf("invalid events[%v].ServiceID('%v'): %w", idx, serviceID, err)
		}
	}

	return nil
}

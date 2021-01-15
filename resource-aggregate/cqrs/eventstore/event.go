package eventstore

import (
	"context"
)

//Event interface over event created by user.
type Event = interface {
	Version() uint64
	EventType() string
}

//EventUnmarshaler provides event.
type EventUnmarshaler = interface {
	Version() uint64
	EventType() string
	AggregateID() string
	GroupID() string
	Unmarshal(v interface{}) error
}

//Iter provides iterator over events from eventstore or eventbus.
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
	dataUnmarshaler func(v interface{}) error
}

func NewLoadedEvent(
	version uint64,
	eventType string,
	aggregateID string,
	groupID string,
	dataUnmarshaler func(v interface{}) error) LoadedEvent {
	return LoadedEvent{
		version:         version,
		eventType:       eventType,
		aggregateID:     aggregateID,
		groupID:         groupID,
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

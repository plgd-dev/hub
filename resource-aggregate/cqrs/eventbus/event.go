package eventbus

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

package maintenance

import (
	"context"
)

// EventStore provides interface over the maintenance functionality for an event store
type EventStore interface {
	Insert(ctx context.Context, task Task) error
	Query(ctx context.Context, limit int, taskHandler TaskHandler) error
	Remove(ctx context.Context, task Task) error
}

// Task used to target a specific db record
type Task struct {
	AggregateID string
	Version     uint64
}

// TaskHandler handles the maintenance db queries
type TaskHandler interface {
	Handle(ctx context.Context, iter Iter) (err error)
}

//Iter provides iterator over maintenance db records
type Iter interface {
	Next(ctx context.Context, task *Task) bool
	Err() error
}

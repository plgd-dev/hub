package eventstore

import (
	"context"
)

// VersionQuery used to load events from version.
type VersionQuery struct {
	GroupID     string //required
	AggregateID string //required
	Version     uint64 //required
}

// SnapshotQuery used to load events from snapshot.
type SnapshotQuery struct {
	GroupID           string //filter by group ID
	AggregateID       string //filter to certain aggregateID, groupID is required
	SnapshotEventType string //required
}

// EventStore provides interface over eventstore. More aggregates can be grouped by groupID,
// but aggregateID of aggregates must be unique against whole DB.
type EventStore interface {
	Save(ctx context.Context, groupID string, aggregateID string, events []Event) (concurrencyException bool, err error)
	SaveSnapshot(ctx context.Context, groupID string, aggregateID string, event Event) (concurrencyException bool, err error)
	LoadUpToVersion(ctx context.Context, queries []VersionQuery, eventHandler Handler) error
	LoadFromVersion(ctx context.Context, queries []VersionQuery, eventHandler Handler) error
	LoadFromSnapshot(ctx context.Context, queries []SnapshotQuery, eventHandler Handler) error
	RemoveUpToVersion(ctx context.Context, queries []VersionQuery) error
}

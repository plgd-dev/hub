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
	GroupID     string //filter by group ID
	AggregateID string //filter to certain aggregateID, groupID is required
}

// Get events with given attributes.
// All filtering options are optional, if none are given then all events are returned,
type GetEventsQuery struct {
	GroupID     string //filter by group ID, optional
	AggregateID string //filter to certain aggregateID, optional
}

type SaveStatus int

const (
	Ok                   SaveStatus = 0  // events were stored
	ConcurrencyException SaveStatus = 1  // events with this version already exists
	SnapshotRequired     SaveStatus = 2  // event store requires aggregated snapshot before applying new event; snapshot shall not contain this new event
	Fail                 SaveStatus = -1 // error occurred
)

// EventStore provides interface over eventstore. More aggregates can be grouped by groupID,
// but aggregateID of aggregates must be unique against whole DB.
type EventStore interface {
	// Get events from the eventstore with timestamp larger than given value
	// If timestamp is <=0 then the argument is ignored.
	GetEvents(ctx context.Context, queries []GetEventsQuery, timestamp int64, eventHandler Handler) error
	// Save save events to eventstore.
	// AggregateID, GroupID and EventType are required.
	// All events within one Save operation shall have the same AggregateID and GroupID.
	// Versions shall be unique and ascend continually.
	// Only first event can be a snapshot.
	Save(ctx context.Context, events ...Event) (status SaveStatus, err error)
	LoadUpToVersion(ctx context.Context, queries []VersionQuery, eventHandler Handler) error
	LoadFromVersion(ctx context.Context, queries []VersionQuery, eventHandler Handler) error
	LoadFromSnapshot(ctx context.Context, queries []SnapshotQuery, eventHandler Handler) error
	RemoveUpToVersion(ctx context.Context, queries []VersionQuery) error
}

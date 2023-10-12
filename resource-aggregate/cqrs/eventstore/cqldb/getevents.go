package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Get events from the eventstore.
func (s *EventStore) GetEvents(ctx context.Context, queries []eventstore.GetEventsQuery, timestamp int64, eventHandler eventstore.Handler) error {
	if len(queries) == 0 {
		return status.Errorf(codes.InvalidArgument, "invalid queries")
	}
	q := make([]eventstore.SnapshotQuery, 0, len(queries))
	for _, query := range queries {
		q = append(q, eventstore.SnapshotQuery(query))
	}
	normalizedQueries := normalizeQueries(q)
	return s.loadFromSnapshot(ctx, normalizedQueries, timestamp, eventHandler)
}

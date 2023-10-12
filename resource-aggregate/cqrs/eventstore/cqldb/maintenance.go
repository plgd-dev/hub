package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/maintenance"
)

// Query retrieves the latest snapshot version per aggregate for thw number of aggregates specified by 'limit'
func (s *EventStore) Query(_ context.Context, _ int, _ maintenance.TaskHandler) error {
	return eventstore.ErrNotSupported
}

// Remove deletes (the latest snapshot version) database record for a given aggregate ID
func (s *EventStore) Remove(_ context.Context, _ maintenance.Task) error {
	// not supported
	return eventstore.ErrNotSupported
}

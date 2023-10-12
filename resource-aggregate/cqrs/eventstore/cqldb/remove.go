package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

// RemoveUpToVersion deletes the aggregated events up to a specific version.
func (s *EventStore) RemoveUpToVersion(_ context.Context, _ []eventstore.VersionQuery) error {
	return eventstore.ErrNotSupported
}

package cqldb

import (
	"context"
	"errors"
	"strings"

	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
)

func getDeviceIDFilter(queries []eventstore.DeleteQuery) string {
	if len(queries) == 0 {
		return ""
	}
	var b strings.Builder
	for _, query := range queries {
		if query.GroupID == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(query.GroupID)
	}
	return b.String()
}

// Delete documents with given group ids
func (s *EventStore) Delete(ctx context.Context, queries []eventstore.DeleteQuery) error {
	deviceIDFilter := getDeviceIDFilter(queries)
	if len(deviceIDFilter) == 0 {
		return errors.New("failed to delete documents: invalid query")
	}

	return s.client.Session().Query("delete from " + s.Table() + " " + cqldb.WhereClause + " " + deviceIDKey + " in (" + deviceIDFilter + ");").WithContext(ctx).Exec()
}

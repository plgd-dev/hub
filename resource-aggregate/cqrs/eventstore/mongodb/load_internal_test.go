package mongodb

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/stretchr/testify/require"
)

func TestUniqueQuery(t *testing.T) {
	type args struct {
		queries []eventstore.SnapshotQuery
		query   eventstore.SnapshotQuery
	}
	test := []struct {
		name string
		args args
		want []eventstore.SnapshotQuery
	}{
		{
			name: "first query",
			args: args{
				queries: nil,
				query:   eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
			},
		},
		{
			name: "same query",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
			},
		},
		{
			name: "two queries",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
				},
				query: eventstore.SnapshotQuery{Types: []string{"type2"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
				{Types: []string{"type2"}},
			},
		},
		{
			name: "most general query",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
				},
				query: eventstore.SnapshotQuery{},
			},
			want: []eventstore.SnapshotQuery{
				{},
			},
		},
		{
			name: "replace a query with more general query",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
					{AggregateID: "1", Types: []string{"type2"}},
					{AggregateID: "2", Types: []string{"type2"}},
				},
				query: eventstore.SnapshotQuery{Types: []string{"type2"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
				{Types: []string{"type2"}},
			},
		},
		{
			name: "replace more queries with more general query with the aggregateID",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
					{AggregateID: "1", Types: []string{"type2"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1"},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1"},
			},
		},
		{
			name: "replace a query with more general query with the aggregateID",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type2", "type1"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
			},
		},
		{
			name: "use general query instead of more specific query with the aggregateID",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{AggregateID: "1", Types: []string{"type1"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1", "type2"}},
			},
			want: []eventstore.SnapshotQuery{
				{AggregateID: "1", Types: []string{"type1"}},
			},
		},
		{
			name: "use general query instead of more specific query",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{Types: []string{"type1"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1", "type2"}},
			},
			want: []eventstore.SnapshotQuery{
				{Types: []string{"type1"}},
			},
		},
		{
			name: "general and specific query with types",
			args: args{
				queries: []eventstore.SnapshotQuery{
					{Types: []string{"type2", "type1"}},
				},
				query: eventstore.SnapshotQuery{AggregateID: "1", Types: []string{"type1"}},
			},
			want: []eventstore.SnapshotQuery{
				{Types: []string{"type2", "type1"}},
				{AggregateID: "1", Types: []string{"type1"}},
			},
		},
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueQuery(tt.args.queries, tt.args.query)
			require.Equal(t, tt.want, got)
		})
	}
}

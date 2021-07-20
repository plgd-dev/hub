package mongodb

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
	"github.com/stretchr/testify/require"
)

func Test_getNormalizedGetEventsFilter(t *testing.T) {
	const groupID1 = "groupID1"
	const aggregateID1 = "aggregateID1"
	const aggregateID2 = "aggregateID2"

	type args struct {
		queries []eventstore.GetEventsQuery
	}
	tests := []struct {
		name string
		args args
		want deviceIdFilter
	}{
		{
			name: "Remove duplicates",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
				},
			},
			want: deviceIdFilter{
				all: false,
				deviceIds: map[string]resourceIdFilter{
					groupID1: {
						all:         false,
						resourceIds: strings.MakeSet(aggregateID1, aggregateID2),
					},
				},
			},
		},
		{
			name: "Absorb aggregate ids",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID: groupID1,
					},
				},
			},
			want: deviceIdFilter{
				all: false,
				deviceIds: map[string]resourceIdFilter{
					groupID1: {
						all: true,
					},
				},
			},
		},
		{
			name: "Absorb group ids",
			args: args{
				queries: []eventstore.GetEventsQuery{
					{
						GroupID:     groupID1,
						AggregateID: aggregateID1,
					},
					{
						GroupID:     groupID1,
						AggregateID: aggregateID2,
					},
					{
						GroupID: "",
					},
				},
			},
			want: deviceIdFilter{
				all: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNormalizedGetEventsFilter(tt.args.queries)
			require.Equal(t, got, tt.want)
		})
	}
}

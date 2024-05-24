package mongodb

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/stretchr/testify/require"
)

func TestConfigurationNormalizeQuery(t *testing.T) {
	tests := []struct {
		name  string
		query *pb.GetConfigurationsRequest
		want  []*pb.IDFilter
	}{
		{
			name:  "nil",
			query: nil,
			want:  nil,
		},
		{
			name:  "empty",
			query: &pb.GetConfigurationsRequest{},
			want:  nil,
		},
		{
			// if the query contains an IDFilter with an empty ID and All set to true, then all other filters are ignored
			name: "all",
			query: &pb.GetConfigurationsRequest{
				IdFilter: []*pb.IDFilter{
					{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
					{Version: &pb.IDFilter_All{All: true}},
				},
			},
			want: nil,
		},
		{
			// if the query contains an IDFilter with All set to true then other filters for the ID are ignored
			name: "remove non-All",
			query: &pb.GetConfigurationsRequest{
				IdFilter: []*pb.IDFilter{
					{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id1", Version: &pb.IDFilter_Value{Value: 1}},
					{Id: "id1", Version: &pb.IDFilter_All{All: true}},
				},
			},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
			},
		},
		{
			name: "remove duplicates",
			query: &pb.GetConfigurationsRequest{
				IdFilter: []*pb.IDFilter{
					{Id: "id1", Version: &pb.IDFilter_All{All: true}},
					{Id: "id1", Version: &pb.IDFilter_All{All: true}},
					{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
					{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
				},
			},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
				{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
			},
		},
		{
			name: "normalize",
			query: &pb.GetConfigurationsRequest{
				IdFilter: []*pb.IDFilter{
					{Id: "id3", Version: &pb.IDFilter_Value{Value: 3}},
					{Id: "id1", Version: &pb.IDFilter_Value{Value: 0}},
					{Id: "id3", Version: &pb.IDFilter_Value{Value: 3}},
					{Id: "id2", Version: &pb.IDFilter_All{All: true}},
					{Id: "id3", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id1", Version: &pb.IDFilter_Value{Value: 1}},
					{Id: "id3", Version: &pb.IDFilter_Value{Value: 1}},
					{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id3", Version: &pb.IDFilter_Value{Value: 2}},
					{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
					{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id3", Version: &pb.IDFilter_Latest{Latest: true}},
					{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
				},
			},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id1", Version: &pb.IDFilter_Value{Value: 0}},
				{Id: "id1", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id2", Version: &pb.IDFilter_All{All: true}},
				{Id: "id3", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id3", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id3", Version: &pb.IDFilter_Value{Value: 2}},
				{Id: "id3", Version: &pb.IDFilter_Value{Value: 3}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.EqualValues(t, tt.want, normalizeIdFilter(tt.query.GetIdFilter()))
		})
	}
}

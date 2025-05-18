package pb_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/stretchr/testify/require"
)

func TestNormalizeIDFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter []*pb.IDFilter
		want   []*pb.IDFilter
	}{
		{
			name:   "nil",
			filter: nil,
			want:   nil,
		},
		{
			// if the query contains an IDFilter with an empty ID and All set to true, then all other filters are ignored
			name: "all",
			filter: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
				{Version: &pb.IDFilter_All{All: true}},
			},
			want: nil,
		},
		{
			// if the query contains an IDFilter with All set to true then other filters for the ID are ignored
			name: "remove non-All",
			filter: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id1", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
			},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
			},
		},
		{
			name: "remove duplicates",
			filter: []*pb.IDFilter{
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
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
				{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 42}},
			},
		},
		{
			name: "normalize",
			filter: []*pb.IDFilter{
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
			require.Equal(t, tt.want, pb.NormalizeIdFilter(tt.filter))
		})
	}
}

func TestIDFilterFromString(t *testing.T) {
	tests := []struct {
		name   string
		filter []string
		want   []*pb.IDFilter
	}{
		{
			name:   "nil",
			filter: nil,
			want:   nil,
		},
		{
			name:   "empty",
			filter: []string{""},
			want:   []*pb.IDFilter{},
		},
		{
			name:   "all",
			filter: []string{"/id1", "/id2/", "id3/", "/id4/all"},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_All{All: true}},
				{Id: "id2", Version: &pb.IDFilter_All{All: true}},
				{Id: "id3", Version: &pb.IDFilter_All{All: true}},
				{Id: "id4", Version: &pb.IDFilter_All{All: true}},
			},
		},
		{
			name:   "latest",
			filter: []string{"/id1/latest", "id2/latest"},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_Latest{Latest: true}},
				{Id: "id2", Version: &pb.IDFilter_Latest{Latest: true}},
			},
		},
		{
			name:   "value",
			filter: []string{"/id1/0", "id2/1", "id3/2", "id4/42"},
			want: []*pb.IDFilter{
				{Id: "id1", Version: &pb.IDFilter_Value{Value: 0}},
				{Id: "id2", Version: &pb.IDFilter_Value{Value: 1}},
				{Id: "id3", Version: &pb.IDFilter_Value{Value: 2}},
				{Id: "id4", Version: &pb.IDFilter_Value{Value: 42}},
			},
		},
		{
			name:   "invalid",
			filter: []string{"id1/fail", "/id2/fail"},
			want:   []*pb.IDFilter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pb.IDFilterFromString(tt.filter))
		})
	}
}

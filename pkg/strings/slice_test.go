package strings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntersection(t *testing.T) {
	type args struct {
		s1 []string
		s2 []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Empty",
			args: args{},
			want: nil,
		},
		{
			name: "Left empty",
			args: args{
				s2: []string{"1"},
			},
			want: nil,
		},
		{
			name: "Right empty",
			args: args{
				s1: []string{"1"},
			},
			want: nil,
		},
		{
			name: "Identical",
			args: args{
				s1: []string{"1", "2", "3"},
				s2: []string{"1", "2", "3"},
			},
			want: []string{"1", "2", "3"},
		},
		{
			name: "Left subset",
			args: args{
				s1: []string{"1", "3"},
				s2: []string{"1", "2", "3"},
			},
			want: []string{"1", "3"},
		},
		{
			name: "Right subset",
			args: args{
				s1: []string{"1", "2", "3"},
				s2: []string{"1", "2"},
			},
			want: []string{"1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Intersection(tt.args.s1, tt.args.s2)
			require.Equal(t, got, tt.want)
		})
	}
}

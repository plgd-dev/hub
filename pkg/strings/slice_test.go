package strings

import (
	"sort"
	"strconv"
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

func isNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func TestSplit(t *testing.T) {
	type args struct {
		s []string
		f SplitFilter
	}
	tests := []struct {
		name      string
		args      args
		wantTrue  []string
		wantFalse []string
	}{
		{
			name: "Empty",
			args: args{
				f: isNumeric,
			},
		},
		{
			name: "All true",
			args: args{
				s: []string{"1", "42", "1337"},
				f: isNumeric,
			},
			wantTrue: []string{"1", "42", "1337"},
		},
		{
			name: "All false",
			args: args{
				s: []string{"a", "bc", "leet"},
				f: isNumeric,
			},
			wantFalse: []string{"a", "bc", "leet"},
		},
		{
			name: "Some true",
			args: args{
				s: []string{"first", "42", "middle", "1337", "last"},
				f: isNumeric,
			},
			wantTrue:  []string{"42", "1337"},
			wantFalse: []string{"first", "middle", "last"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTrue, gotFalse := Split(tt.args.s, tt.args.f)
			require.Equal(t, gotTrue, tt.wantTrue)
			require.Equal(t, gotFalse, tt.wantFalse)
		})
	}
}

func TestUnique(t *testing.T) {
	type args struct {
		s []string
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
			name: "Single",
			args: args{
				s: []string{"a", "a", "a"},
			},
			want: []string{"a"},
		},
		{
			name: "Single",
			args: args{
				s: []string{"a", "a", "a"},
			},
			want: []string{"a"},
		},
		{
			name: "Multiple",
			args: args{
				s: []string{"a", "a", "a", "b", "b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.args.s)
			sort.Strings(got)
			sort.Strings(tt.want)
			require.Equal(t, got, tt.want)
		})
	}
}

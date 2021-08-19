package strings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeSortedSlice(t *testing.T) {
	type args struct {
		v []string
	}
	tests := []struct {
		name string
		args args
		want SortedSlice
	}{
		{
			name: "Create from single element",
			args: args{v: []string{"1"}},
			want: SortedSlice([]string{"1"}),
		},
		{
			name: "Create from not sorted",
			args: args{v: []string{"3", "2", "1"}},
			want: SortedSlice([]string{"1", "2", "3"}),
		},
		{
			name: "Create from not unique",
			args: args{v: []string{"1", "1", "1"}},
			want: SortedSlice([]string{"1"}),
		},
		{
			name: "Create from not unique, not sorted",
			args: args{v: []string{"3", "2", "1", "3", "2", "3", "3", "3", "4"}},
			want: SortedSlice([]string{"1", "2", "3", "4"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeSortedSlice(tt.args.v)

			require.Equal(t, tt.want, got)
		})
	}
}

func TestInsert(t *testing.T) {
	type args struct {
		slice SortedSlice
		elems []string
	}
	tests := []struct {
		name string
		args args
		want SortedSlice
	}{
		{
			name: "Empty insert",
			args: args{
				slice: MakeSortedSlice([]string{"e", "f", "g"}),
			},
			want: SortedSlice([]string{"e", "f", "g"}),
		},
		{
			name: "Insert single (start)",
			args: args{
				slice: MakeSortedSlice([]string{"e", "f", "g"}),
				elems: []string{"a"},
			},
			want: SortedSlice([]string{"a", "e", "f", "g"}),
		},
		{
			name: "Insert single (middle)",
			args: args{
				slice: MakeSortedSlice([]string{"d", "r"}),
				elems: []string{"m"},
			},
			want: SortedSlice([]string{"d", "m", "r"}),
		},
		{
			name: "Insert single (end)",
			args: args{
				slice: MakeSortedSlice([]string{"e", "f", "g"}),
				elems: []string{"z"},
			},
			want: SortedSlice([]string{"e", "f", "g", "z"}),
		},
		{
			name: "Insert duplicate",
			args: args{
				slice: MakeSortedSlice([]string{"a", "b", "c"}),
				elems: []string{"a", "b", "c"},
			},
			want: SortedSlice([]string{"a", "b", "c"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Insert(tt.args.slice, tt.args.elems...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestContains(t *testing.T) {
	type args struct {
		slice SortedSlice
		s     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Found",
			args: args{
				slice: SortedSlice{"1", "2", "3"},
				s:     "1",
			},
			want: true,
		},
		{
			name: "Not found",
			args: args{
				slice: SortedSlice{"4", "2", "4242", "042", " 42", "42 "},
				s:     "42",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Contains(tt.args.slice, tt.args.s))
		})
	}
}

func TestRemove(t *testing.T) {
	type args struct {
		slice SortedSlice
		elems []string
	}
	tests := []struct {
		name string
		args args
		want SortedSlice
	}{
		{
			name: "Empty",
			args: args{
				slice: MakeSortedSlice([]string{}),
			},
			want: SortedSlice([]string{}),
		},
		{
			name: "Not found",
			args: args{
				slice: MakeSortedSlice([]string{"a", "bb", "ccc"}),
				elems: []string{"a1", "b2", "c3"},
			},
			want: SortedSlice([]string{"a", "bb", "ccc"}),
		},
		{
			name: "Single",
			args: args{
				slice: MakeSortedSlice([]string{"a", "b", "c"}),
				elems: []string{"a"},
			},
			want: SortedSlice([]string{"b", "c"}),
		},
		{
			name: "Some",
			args: args{
				slice: MakeSortedSlice([]string{"a", "bb", "ccc", "dddd", "eeeee"}),
				elems: []string{"bb", "dddd"},
			},
			want: SortedSlice([]string{"a", "ccc", "eeeee"}),
		},
		{
			name: "all",
			args: args{
				slice: MakeSortedSlice([]string{"a", "bb", "ccc", "dddd", "eeeee"}),
				elems: []string{"a", "bb", "ccc", "dddd", "eeeee"},
			},
			want: SortedSlice([]string{}),
		},
		{
			name: "mixed",
			args: args{
				slice: MakeSortedSlice([]string{"a", "bb", "ccc", "dddd"}),
				elems: []string{"40", "bb", "41", "ccc", "42"},
			},
			want: SortedSlice([]string{"a", "dddd"}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Remove(tt.args.slice, tt.args.elems...)
			require.Equal(t, tt.want, got)
		})
	}
}

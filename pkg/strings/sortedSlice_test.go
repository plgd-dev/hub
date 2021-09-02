package strings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortedSlice_MakeSortedSlice(t *testing.T) {
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

func TestSortedSlice_Insert(t *testing.T) {
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
			got := tt.args.slice.Insert(tt.args.elems...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSortedSlice_Contains(t *testing.T) {
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
			require.Equal(t, tt.want, tt.args.slice.Contains(tt.args.s))
		})
	}
}

func TestSortedSlice_Remove(t *testing.T) {
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
			got := tt.args.slice.Remove(tt.args.elems...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSortedSlice_Difference(t *testing.T) {
	type args struct {
		first  SortedSlice
		second SortedSlice
	}
	tests := []struct {
		name string
		args args
		want SortedSlice
	}{
		{
			name: "Empty first",
			args: args{
				first:  SortedSlice{},
				second: SortedSlice{"a", "b", "c"},
			},
			want: nil,
		},
		{
			name: "Empty second",
			args: args{
				first:  SortedSlice{"a", "b", "c"},
				second: SortedSlice{},
			},
			want: SortedSlice{"a", "b", "c"},
		},
		{
			name: "Identity",
			args: args{
				first:  SortedSlice{"a", "b", "c"},
				second: SortedSlice{"a", "b", "c"},
			},
			want: nil,
		},
		{
			name: "Subset (1)",
			args: args{
				first:  SortedSlice{"a", "b", "c", "d", "e"},
				second: SortedSlice{"a", "b", "c"},
			},
			want: SortedSlice{"d", "e"},
		},
		{
			name: "Subset (2)",
			args: args{
				first:  SortedSlice{"a", "b", "c", "d", "e"},
				second: SortedSlice{"d", "e"},
			},
			want: SortedSlice{"a", "b", "c"},
		},
		{
			name: "Superset",
			args: args{
				first:  SortedSlice{"b", "d", "e"},
				second: SortedSlice{"a", "b", "c", "d", "e"},
			},
			want: nil,
		},
		{
			name: "Mixed",
			args: args{
				first:  SortedSlice{"a", "b", "c", "x", "y"},
				second: SortedSlice{"a", "d", "e", "y", "z"},
			},
			want: SortedSlice{"b", "c", "x"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.first.Difference(tt.args.second)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSortedSlice_Intersection(t *testing.T) {
	type args struct {
		second SortedSlice
	}
	tests := []struct {
		name  string
		slice SortedSlice
		args  args
		want  SortedSlice
	}{
		{
			name:  "Empty",
			slice: nil,
			args:  args{},
			want:  nil,
		},
		{
			name:  "Left empty",
			slice: nil,
			args: args{
				second: SortedSlice{"1"},
			},
			want: nil,
		},
		{
			name:  "Right empty",
			slice: SortedSlice{"1"},
			want:  nil,
		},
		{
			name:  "Identical",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				second: SortedSlice{"1", "2", "3"},
			},
			want: SortedSlice{"1", "2", "3"},
		},
		{
			name:  "Left subset",
			slice: SortedSlice{"1", "3"},
			args: args{
				second: SortedSlice{"1", "2", "3"},
			},
			want: SortedSlice{"1", "3"},
		},
		{
			name:  "Right subset",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				second: SortedSlice{"1", "2"},
			},
			want: SortedSlice{"1", "2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.slice.Intersection(tt.args.second)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestSortedSlice_Equal(t *testing.T) {
	type args struct {
		second SortedSlice
	}
	tests := []struct {
		name  string
		slice SortedSlice
		args  args
		want  bool
	}{
		{
			name:  "Empty",
			slice: nil,
			args:  args{},
			want:  true,
		},
		{
			name:  "First slice longer",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				second: SortedSlice{"1", "2"},
			},
			want: false,
		},
		{
			name:  "Second slice longer",
			slice: SortedSlice{"1", "2"},
			args: args{
				second: SortedSlice{"1", "2", "3"},
			},
			want: false,
		},
		{
			name:  "Same length (different)",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				second: SortedSlice{"1", "2", "42"},
			},
			want: false,
		},
		{
			name:  "Same length (equal)",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				second: SortedSlice{"1", "2", "3"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.slice.Equal(tt.args.second)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestSortedSlice_IsSuperslice(t *testing.T) {
	type args struct {
		s SortedSlice
	}
	tests := []struct {
		name  string
		slice SortedSlice
		args  args
		want  bool
	}{
		{
			name:  "Both empty",
			slice: nil,
			args:  args{},
			want:  true,
		},
		{
			name:  "Left empty",
			slice: nil,
			args: args{
				s: SortedSlice{"1", "2", "3"},
			},
			want: false,
		},
		{
			name:  "Right empty",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				s: nil,
			},
			want: true,
		},
		{
			name:  "Single element",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				s: SortedSlice{"2"},
			},
			want: true,
		},
		{
			name:  "Equal",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				s: SortedSlice{"1", "2", "3"},
			},
			want: true,
		},
		{
			name:  "Subset",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				s: SortedSlice{"0", "1", "2", "3"},
			},
			want: false,
		},
		{
			name:  "Superset",
			slice: SortedSlice{"0", "1", "2", "3", "4", "5"},
			args: args{
				s: SortedSlice{"0", "2", "3", "5"},
			},
			want: true,
		},
		{
			name:  "Not superset",
			slice: SortedSlice{"1", "2", "3"},
			args: args{
				s: SortedSlice{"1", "42"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.slice.IsSuperslice(tt.args.s)
			require.Equal(t, got, tt.want)
		})
	}
}

package strings_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/stretchr/testify/require"
)

func TestUniqueStable(t *testing.T) {
	tests := []struct {
		name string
		s    []string
		want []string
	}{
		{
			name: "No duplicates",
			s:    []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "With duplicates",
			s:    []string{"a", "b", "a", "c", "b"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "Empty slice",
			s:    []string{},
			want: []string{},
		},
		{
			name: "All duplicates",
			s:    []string{"a", "a", "a"},
			want: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strings.UniqueStable(tt.s)
			require.Equal(t, tt.want, got)
		})
	}
}

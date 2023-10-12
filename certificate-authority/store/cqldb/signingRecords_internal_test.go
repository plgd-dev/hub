package cqldb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimaryKeysValuesUnique(t *testing.T) {
	tests := []struct {
		name string
		p    primaryKeysValues
		want primaryKeysValues
	}{
		{
			name: "no duplicates",
			p: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
			},
			want: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
			},
		},
		{
			name: "one duplicate",
			p: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
				{id: "2"},
			},
			want: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
			},
		},
		{
			name: "multiple duplicates",
			p: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
				{id: "4"},
				{id: "3"},
				{id: "2"},
				{id: "3"},
			},
			want: primaryKeysValues{
				{id: "1"},
				{id: "2"},
				{id: "3"},
				{id: "4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.unique()
			require.Equal(t, tt.want, got)
		})
	}
}

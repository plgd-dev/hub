//go:build test
// +build test

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseUnpublishQueryString(t *testing.T) {
	tests := []struct {
		name        string
		queries     []string
		expectedID  string
		expectedIns []int64
		wantErr     bool
	}{
		{
			name:        "ValidQueries",
			queries:     []string{"di=device1", "ins=1", "ins=2"},
			expectedID:  "device1",
			expectedIns: []int64{1, 2},
		},
		{
			name:        "MultipleInsInOneQuery",
			queries:     []string{"di=device1&ins=1&ins=2"},
			expectedID:  "device1",
			expectedIns: []int64{1, 2},
		},
		{
			name:    "InvalidIns",
			queries: []string{"di=device1", "ins=abc"},
			wantErr: true,
		},
		{
			name:        "One of the queries is invalid",
			queries:     []string{"di=device1", "ins=1", "invalid_query"},
			expectedID:  "device1",
			expectedIns: []int64{1},
		},
		{
			name:    "MultipleDeviceIDs",
			queries: []string{"di=device1&di=device2"},
			wantErr: true,
		},
		{
			name:    "DeviceIDandInsAreNotSet",
			queries: []string{"invalidQuery=123"},
			wantErr: true,
		},
		{
			name:    "EmptyQueries",
			queries: []string{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deviceID, instanceIDs, err := parseUnpublishQueryString(test.queries)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, test.expectedID, deviceID)
			require.Equal(t, test.expectedIns, instanceIDs)
		})
	}
}

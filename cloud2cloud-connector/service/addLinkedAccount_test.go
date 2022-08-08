package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name    string
		args    args
		wantLen int
	}{
		{
			name: "empty",
			args: args{
				n: 0,
			},
		},
		{
			name: "not empty",
			args: args{
				n: 16,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateRandomString(tt.args.n)
			require.NoError(t, err)
			if tt.args.n == 0 {
				require.Empty(t, got)
				return
			}
			require.NotEmpty(t, got)
		})
	}
}

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchPrefixAndSplitURIPath(t *testing.T) {
	type args struct {
		requestURI string
		prefix     string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty",
			args: args{
				requestURI: "",
			},
			want: nil,
		},
		{
			name: "no prefix",
			args: args{
				requestURI: "test",
			},
			want: nil,
		},
		{
			name: "prefix",
			args: args{
				requestURI: "test",
				prefix:     "test",
			},
			want: nil,
		},
		{
			name: "valid",
			args: args{
				requestURI: "/1/2/3",
				prefix:     "/1",
			},
			want: []string{"2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPrefixAndSplitURIPath(tt.args.requestURI, tt.args.prefix)
			require.Equal(t, tt.want, got)
		})
	}
}

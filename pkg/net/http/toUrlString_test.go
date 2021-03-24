package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToURLString(t *testing.T) {
	type args struct {
		scheme string
		host   string
		path   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "a://b/c",
			args: args{
				scheme: "a",
				host:   "b",
				path:   "c",
			},
			want: "a://b/c",
		},
		{
			name: "a://b/%",
			args: args{
				scheme: "a",
				host:   "b",
				path:   "%",
			},
			want: "a://b/%25",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToURLString(tt.args.scheme, tt.args.host, tt.args.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

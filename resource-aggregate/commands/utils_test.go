package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResourceIdFromString(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name string
		args args
		want *ResourceId
	}{
		{
			name: "Nil",
			args: args{
				v: "",
			},
			want: nil,
		},
		{
			name: "Simple",
			args: args{
				v: "deviceId/href",
			},
			want: &ResourceId{
				DeviceId: "deviceId",
				Href:     "/href",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResourceIdFromString(tt.args.v)
			require.True(t, proto.Equal(tt.want, got))
		})
	}
}

package events_test

import (
	"testing"

	"github.com/plgd-dev/go-coap/v2/message"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventResourceUpdated events.ResourceUpdated = events.ResourceUpdated{
	ResourceId: &commands.ResourceId{
		DeviceId: "dev1",
		Href:     "/dev1",
	},
	Status: commands.Status_ACCEPTED,
	Content: &commands.Content{
		Data:              []byte{'t', 'e', 'x', 't'},
		ContentType:       "text",
		CoapContentFormat: int32(message.TextPlain),
	},
	AuditContext: &commands.AuditContext{
		UserId:        "501",
		CorrelationId: "1",
	},
	EventMetadata: &events.EventMetadata{
		Version:      42,
		Timestamp:    12345,
		ConnectionId: "con1",
		Sequence:     1,
	},
}

func TestResourceUpdated_CopyData(t *testing.T) {
	type args struct {
		event *events.ResourceUpdated
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventResourceUpdated,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceUpdated
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestResourceUpdated_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceUpdated
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceUpdated{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceUpdated,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.args.event.CheckInitialized())
		})
	}
}

package events_test

import (
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventResourceChanged events.ResourceChanged = events.ResourceChanged{
	ResourceId: &commands.ResourceId{
		DeviceId: dev1,
		Href:     "/dev1",
	},
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
	Status:        commands.Status_ACCEPTED,
	ResourceTypes: []string{"type1", "type2"},
}

func TestResourceChangedCopyData(t *testing.T) {
	type args struct {
		event *events.ResourceChanged
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventResourceChanged,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceChanged
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestResourceChanged_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceChanged
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceChanged{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceChanged,
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

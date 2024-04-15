package events_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/interfaces"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventResourceDeletePending events.ResourceDeletePending = events.ResourceDeletePending{
	ResourceId: &commands.ResourceId{
		DeviceId: dev1,
		Href:     "/dev1",
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
	ResourceInterface: interfaces.OC_IF_BASELINE,
	ResourceTypes:     []string{"type1", "type2"},
}

func TestResourceDeletePendingCopyData(t *testing.T) {
	type args struct {
		event *events.ResourceDeletePending
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventResourceDeletePending,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceDeletePending
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestResourceDeletePending_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceDeletePending
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceDeletePending{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceDeletePending,
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

package events_test

import (
	"testing"

	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventResourceRetrievePending events.ResourceRetrievePending = events.ResourceRetrievePending{
	ResourceId: &commands.ResourceId{
		DeviceId: dev1,
		Href:     "/dev1",
	},
	ResourceInterface: "if1",
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
	ResourceTypes: []string{"type1", "type2"},
}

func TestResourceRetrievePendingCopyData(t *testing.T) {
	type args struct {
		event *events.ResourceRetrievePending
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventResourceRetrievePending,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceRetrievePending
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestResourceRetrievePending_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceRetrievePending
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceRetrievePending{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceRetrievePending,
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

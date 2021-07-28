package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventDeviceMetadataUpdatePending events.DeviceMetadataUpdatePending = events.DeviceMetadataUpdatePending{
	DeviceId: "dev1",
	UpdatePending: &events.DeviceMetadataUpdatePending_ShadowSynchronization{
		ShadowSynchronization: commands.ShadowSynchronization_ENABLED,
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

func TestDeviceMetadataUpdatePending_CopyData(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataUpdatePending
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventDeviceMetadataUpdatePending,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.DeviceMetadataUpdatePending
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestDeviceMetadataUpdatePending_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataUpdatePending
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.DeviceMetadataUpdatePending{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventDeviceMetadataUpdatePending,
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

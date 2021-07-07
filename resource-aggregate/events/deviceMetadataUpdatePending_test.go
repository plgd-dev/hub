package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestDeviceMetadataUpdatePending_CopyData(t *testing.T) {

	evt := events.DeviceMetadataUpdatePending{
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
				event: &evt,
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

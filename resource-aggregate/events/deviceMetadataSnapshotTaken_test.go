package events_test

import (
	"testing"

	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestDeviceMetadataSnapshotTaken_CopyData(t *testing.T) {
	evt := events.DeviceMetadataSnapshotTaken{
		DeviceId: "dev1",
		DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
			DeviceId: "dev1",
			Status: &commands.ConnectionStatus{
				Value:      commands.ConnectionStatus_ONLINE,
				ValidUntil: 12345,
			},
			ShadowSynchronization: commands.ShadowSynchronization_ENABLED,
			AuditContext: &commands.AuditContext{
				UserId:        "501",
				CorrelationId: "0",
			},
			EventMetadata: &events.EventMetadata{
				Version:      1,
				Timestamp:    42,
				ConnectionId: "con1",
				Sequence:     1,
			},
		},
		UpdatePendings: []*events.DeviceMetadataUpdatePending{
			{
				DeviceId: "dev1",
			},
		},
		EventMetadata: &events.EventMetadata{
			Version:      2,
			Timestamp:    43,
			ConnectionId: "con2",
			Sequence:     2,
		},
	}
	type args struct {
		event *events.DeviceMetadataSnapshotTaken
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
			var e events.DeviceMetadataSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

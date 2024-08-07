package events_test

import (
	"testing"

	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventDeviceMetadataUpdated events.DeviceMetadataUpdated = events.DeviceMetadataUpdated{
	DeviceId: dev1,
	Connection: &commands.Connection{
		Status: commands.Connection_ONLINE,
	},
	TwinEnabled: true,
	AuditContext: &commands.AuditContext{
		UserId:        "501",
		CorrelationId: "0",
	},
	EventMetadata: &events.EventMetadata{
		Version:      42,
		Timestamp:    12345,
		ConnectionId: "con1",
		Sequence:     1,
	},
}

func TestDeviceMetadataUpdated_Equal(t *testing.T) {
	type fields struct {
		Connection   *commands.Connection
		TwinEnabled  bool
		AuditContext *commands.AuditContext
	}
	type args struct {
		upd *events.DeviceMetadataUpdated
	}

	upd := &testEventDeviceMetadataUpdated
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Identity",
			fields: fields{
				Connection:   upd.GetConnection(),
				TwinEnabled:  upd.GetTwinEnabled(),
				AuditContext: upd.GetAuditContext(),
			},
			args: args{upd},
			want: true,
		},
		{
			name: "Changed Connection.Value",
			fields: fields{
				Connection: &commands.Connection{
					Status: commands.Connection_OFFLINE,
				},
				TwinEnabled:  upd.GetTwinEnabled(),
				AuditContext: upd.GetAuditContext(),
			},
			args: args{upd},
			want: false,
		},
		{
			name: "Changed TwinSynchronization",
			fields: fields{
				Connection:   upd.GetConnection(),
				TwinEnabled:  false,
				AuditContext: upd.GetAuditContext(),
			},
			args: args{upd},
			want: false,
		},
		{
			name: "Changed AuditContext.UserId",
			fields: fields{
				Connection:  upd.GetConnection(),
				TwinEnabled: upd.GetTwinEnabled(),
				AuditContext: &commands.AuditContext{
					UserId:        upd.GetAuditContext().GetUserId() + "0",
					CorrelationId: upd.GetAuditContext().GetCorrelationId(),
				},
			},
			args: args{upd},
			want: false,
		},
		{
			name: "Changed AuditContext.CorrelationId",
			fields: fields{
				Connection:  upd.GetConnection(),
				TwinEnabled: upd.GetTwinEnabled(),
				AuditContext: &commands.AuditContext{
					UserId:        upd.GetAuditContext().GetUserId(),
					CorrelationId: upd.GetAuditContext().GetCorrelationId() + "0",
				},
			},
			args: args{upd},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &events.DeviceMetadataUpdated{
				Connection:   tt.fields.Connection,
				TwinEnabled:  tt.fields.TwinEnabled,
				AuditContext: tt.fields.AuditContext,
			}
			got := e.Equal(tt.args.upd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeviceMetadataUpdated_CopyData(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataUpdated
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventDeviceMetadataUpdated,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.DeviceMetadataUpdated
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestDeviceMetadataUpdated_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataUpdated
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.DeviceMetadataUpdated{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventDeviceMetadataUpdated,
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

package events_test

import (
	"testing"

	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestDeviceMetadataUpdated_Equal(t *testing.T) {
	type fields struct {
		Status                *commands.ConnectionStatus
		ShadowSynchronization commands.ShadowSynchronization
		AuditContext          *commands.AuditContext
	}
	type args struct {
		upd *events.DeviceMetadataUpdated
	}

	upd := events.DeviceMetadataUpdated{
		Status: &commands.ConnectionStatus{
			Value:      commands.ConnectionStatus_ONLINE,
			ValidUntil: 0,
		},
		ShadowSynchronization: commands.ShadowSynchronization_ENABLED,
		AuditContext: &commands.AuditContext{
			UserId:        "501",
			CorrelationId: "0",
		},
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Identity",
			fields: fields{
				Status:                upd.Status,
				ShadowSynchronization: upd.ShadowSynchronization,
				AuditContext:          upd.AuditContext,
			},
			args: args{&upd},
			want: true,
		},
		{
			name: "Changed Status.Value",
			fields: fields{
				Status: &commands.ConnectionStatus{
					Value:      commands.ConnectionStatus_OFFLINE,
					ValidUntil: upd.Status.ValidUntil,
				},
				ShadowSynchronization: upd.ShadowSynchronization,
				AuditContext:          upd.AuditContext,
			},
			args: args{&upd},
			want: false,
		},
		{
			name: "Changed Status.ValidUntil",
			fields: fields{
				Status: &commands.ConnectionStatus{
					Value:      upd.Status.Value,
					ValidUntil: upd.Status.ValidUntil + 1,
				},
				ShadowSynchronization: upd.ShadowSynchronization,
				AuditContext:          upd.AuditContext,
			},
			args: args{&upd},
			want: false,
		},
		{
			name: "Changed ShadowSynchronization",
			fields: fields{
				Status:                upd.Status,
				ShadowSynchronization: commands.ShadowSynchronization_DISABLED,
				AuditContext:          upd.AuditContext,
			},
			args: args{&upd},
			want: false,
		},
		{
			name: "Changed AuditContext.UserId",
			fields: fields{
				Status:                upd.Status,
				ShadowSynchronization: upd.ShadowSynchronization,
				AuditContext: &commands.AuditContext{
					UserId:        "502",
					CorrelationId: upd.AuditContext.CorrelationId,
				},
			},
			args: args{&upd},
			want: false,
		},
		{
			name: "Changed AuditContext.CorrelationId",
			fields: fields{
				Status:                upd.Status,
				ShadowSynchronization: upd.ShadowSynchronization,
				AuditContext: &commands.AuditContext{
					UserId:        upd.AuditContext.UserId,
					CorrelationId: "1",
				},
			},
			args: args{&upd},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &events.DeviceMetadataUpdated{
				Status:                tt.fields.Status,
				ShadowSynchronization: tt.fields.ShadowSynchronization,
				AuditContext:          tt.fields.AuditContext,
			}
			got := e.Equal(tt.args.upd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeviceMetadataUpdated_CopyData(t *testing.T) {
	evt := events.DeviceMetadataUpdated{
		DeviceId: "dev1",
		Status: &commands.ConnectionStatus{
			Value:      commands.ConnectionStatus_ONLINE,
			ValidUntil: 12345,
		},
		ShadowSynchronization: commands.ShadowSynchronization_UNSET,
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
		event *events.DeviceMetadataUpdated
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
			var e events.DeviceMetadataUpdated
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

package events_test

import (
	"testing"

	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
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

package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	res := events.ResourceChanged{
		Content: &commands.Content{
			Data:              []byte{'{', '}'},
			ContentType:       "json",
			CoapContentFormat: int32(message.AppJSON),
		},
		AuditContext: &commands.AuditContext{
			UserId: "501",
		},
		Status: commands.Status_OK,
	}

	resWithChangedContent := events.ResourceChanged{
		Content: &commands.Content{
			Data:              []byte{'t', 'e', 'x', 't'},
			ContentType:       "text",
			CoapContentFormat: int32(message.TextPlain),
		},
		AuditContext: res.AuditContext,
		Status:       res.Status,
	}

	resWithChangedAuditContext := events.ResourceChanged{
		Content: res.Content,
		AuditContext: &commands.AuditContext{
			UserId: "502",
		},
		Status: res.Status,
	}

	resWithChangedStatus := events.ResourceChanged{
		Content:      res.Content,
		AuditContext: res.AuditContext,
		Status:       commands.Status_ERROR,
	}

	type args struct {
		current *events.ResourceChanged
		changed *events.ResourceChanged
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Changed content",
			args: args{
				current: &res,
				changed: &resWithChangedContent,
			},
			want: false,
		},
		{
			name: "Changed audit context",
			args: args{
				current: &res,
				changed: &resWithChangedAuditContext,
			},
			want: false,
		},
		{
			name: "Changed status",
			args: args{
				current: &res,
				changed: &resWithChangedStatus,
			},
			want: false,
		},
		{
			name: "Identical",
			args: args{
				current: &res,
				changed: &res,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := events.Equal(tt.args.current, tt.args.changed)
			assert.Equal(t, tt.want, got)
		})
	}
}

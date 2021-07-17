package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

func TestResourceStateSnapshotTaken_CopyData(t *testing.T) {
	evt := events.ResourceStateSnapshotTaken{
		ResourceId: &commands.ResourceId{
			DeviceId: "dev1",
			Href:     "/dev1",
		},
		LatestResourceChange: &events.ResourceChanged{
			ResourceId: &commands.ResourceId{
				DeviceId: "devLatest",
				Href:     "/devLatest",
			},
			Content: &commands.Content{},
		},
		ResourceCreatePendings: []*events.ResourceCreatePending{
			{
				ResourceId: &commands.ResourceId{
					DeviceId: "devCreate",
					Href:     "/devCreate",
				},
			},
		},
		ResourceRetrievePendings: []*events.ResourceRetrievePending{
			{
				ResourceId: &commands.ResourceId{
					DeviceId: "devRetrieve",
					Href:     "/devRetrieve",
				},
			},
		},
		ResourceUpdatePendings: []*events.ResourceUpdatePending{
			{
				ResourceId: &commands.ResourceId{
					DeviceId: "devUpdate",
					Href:     "/devUpdate",
				},
			},
		},
		ResourceDeletePendings: []*events.ResourceDeletePending{
			{
				ResourceId: &commands.ResourceId{
					DeviceId: "devDelete",
					Href:     "/devDelete",
				},
			},
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
		event *events.ResourceStateSnapshotTaken
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
			var e events.ResourceStateSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

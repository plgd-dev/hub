package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type iterator struct {
	idx    int
	events []eventstore.EventUnmarshaler
}

func newIterator(events []eventstore.EventUnmarshaler) *iterator {
	return &iterator{
		events: events,
	}
}

func (i *iterator) Next(context.Context) (eventstore.EventUnmarshaler, bool) {
	if i.idx < len(i.events) {
		e := i.events[i.idx]
		i.idx++
		return e, true
	}
	return nil, false
}

func (i *iterator) Err() error {
	return nil
}

func TestResourceStateSnapshotTaken_Handle(t *testing.T) {
	type args struct {
		events *iterator
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "createPending, created",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceCreatePending(commands.NewResourceID("a", "/a"), &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID"), time.Now().Add(-time.Second)),
					test.MakeResourceCreated(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "retrievePending, retrieved",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceRetrievePending(commands.NewResourceID("a", "/a"), "", events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "1", "userID"), time.Now().Add(-time.Second)),
					test.MakeResourceRetrieved(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "1", "userID")),
				}),
			},
		},
		{
			name: "updatePending, updated",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceUpdatePending(commands.NewResourceID("a", "/a"), &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "2", "userID"), time.Now().Add(-time.Second)),
					test.MakeResourceUpdated(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "2", "userID")),
				}),
			},
		},
		{
			name: "retrievePending, retrieved",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceDeletePending(commands.NewResourceID("a", "/a"), events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "3", "userID"), time.Now().Add(-time.Second)),
					test.MakeResourceDeleted(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "3", "userID")),
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewResourceStateSnapshotTaken()
			err := e.Handle(context.TODO(), tt.args.events)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

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
		AuditContext: res.GetAuditContext(),
		Status:       res.GetStatus(),
	}

	resWithChangedAuditContext := events.ResourceChanged{
		Content: res.GetContent(),
		AuditContext: &commands.AuditContext{
			UserId: "502",
		},
		Status: res.GetStatus(),
	}

	resWithChangedStatus := events.ResourceChanged{
		Content:      res.GetContent(),
		AuditContext: res.GetAuditContext(),
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
			got := tt.args.current.Equal(tt.args.changed)
			assert.Equal(t, tt.want, got)
		})
	}
}

var testEventResourceStateSnapshotTaken events.ResourceStateSnapshotTaken = events.ResourceStateSnapshotTaken{
	ResourceId: &commands.ResourceId{
		DeviceId: dev1,
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

func TestResourceStateSnapshotTaken_CopyData(t *testing.T) {
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
				event: &testEventResourceStateSnapshotTaken,
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

func TestResourceStateSnapshotTaken_CheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceStateSnapshotTaken
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceStateSnapshotTaken{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceStateSnapshotTaken,
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

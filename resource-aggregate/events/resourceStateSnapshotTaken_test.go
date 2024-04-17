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

func TestResourceStateSnapshotTakenResourceTypes(t *testing.T) {
	const (
		href     = "/a"
		deviceID = "a"
		hubID    = "hubID"
		userID   = "userID"
	)
	resourceTypes := []string{"type1", "type2"}

	e := events.NewResourceStateSnapshotTaken()
	require.Empty(t, e.Types())
	err := e.Handle(context.TODO(), newIterator([]eventstore.EventUnmarshaler{test.MakeResourceChangedEvent(commands.NewResourceID(deviceID, href), &commands.Content{}, events.MakeEventMeta("", 0, 0, hubID), commands.NewAuditContext(userID, "0", userID), resourceTypes)}))
	require.NoError(t, err)
	require.Equal(t, resourceTypes, e.Types())
	nextEvents := newIterator([]eventstore.EventUnmarshaler{
		test.MakeResourceCreatePending(commands.NewResourceID(deviceID, href), &commands.Content{}, events.MakeEventMeta("", 0, 1, hubID), commands.NewAuditContext(userID, "0", userID), time.Now().Add(-time.Second), resourceTypes),
		test.MakeResourceCreated(commands.NewResourceID(deviceID, href), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 2, hubID), commands.NewAuditContext(userID, "0", userID), resourceTypes),
		test.MakeResourceRetrievePending(commands.NewResourceID(deviceID, href), "", events.MakeEventMeta("", 0, 3, hubID), commands.NewAuditContext(userID, "1", userID), time.Now().Add(-time.Second), resourceTypes),
		test.MakeResourceRetrieved(commands.NewResourceID(deviceID, href), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 4, hubID), commands.NewAuditContext(userID, "1", userID), resourceTypes),
		test.MakeResourceUpdatePending(commands.NewResourceID(deviceID, href), &commands.Content{}, events.MakeEventMeta("", 0, 5, hubID), commands.NewAuditContext(userID, "2", userID), time.Now().Add(-time.Second), resourceTypes),
		test.MakeResourceUpdated(commands.NewResourceID(deviceID, href), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 6, hubID), commands.NewAuditContext(userID, "2", userID), resourceTypes),
		test.MakeResourceDeletePending(commands.NewResourceID(deviceID, href), events.MakeEventMeta("", 0, 7, hubID), commands.NewAuditContext(userID, "3", userID), time.Now().Add(-time.Second), resourceTypes),
		test.MakeResourceDeleted(commands.NewResourceID(deviceID, href), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 8, hubID), commands.NewAuditContext(userID, "3", userID), resourceTypes),
	})
	err = e.Handle(context.TODO(), nextEvents)
	require.NoError(t, err)
	require.Equal(t, resourceTypes, e.Types())
	resourceTypes = append(resourceTypes, "type3")
	err = e.Handle(context.TODO(), newIterator([]eventstore.EventUnmarshaler{test.MakeResourceChangedEvent(commands.NewResourceID(deviceID, href), &commands.Content{}, events.MakeEventMeta("", 1, 9, hubID), commands.NewAuditContext(userID, "0", userID), resourceTypes)}))
	require.NoError(t, err)
	require.Equal(t, resourceTypes, e.Types())
}

func TestResourceStateSnapshotTakenHandle(t *testing.T) {
	resourceTypes := []string{"type1", "type2"}
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
					test.MakeResourceCreatePending(commands.NewResourceID("a", "/a"), &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID"), time.Now().Add(-time.Second), resourceTypes),
					test.MakeResourceCreated(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID"), resourceTypes),
				}),
			},
		},
		{
			name: "retrievePending, retrieved",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceRetrievePending(commands.NewResourceID("a", "/a"), "", events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "1", "userID"), time.Now().Add(-time.Second), resourceTypes),
					test.MakeResourceRetrieved(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "1", "userID"), resourceTypes),
				}),
			},
		},
		{
			name: "updatePending, updated",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceUpdatePending(commands.NewResourceID("a", "/a"), &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "2", "userID"), time.Now().Add(-time.Second), resourceTypes),
					test.MakeResourceUpdated(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "2", "userID"), resourceTypes),
				}),
			},
		},
		{
			name: "deletePending, deleted",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeResourceDeletePending(commands.NewResourceID("a", "/a"), events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "3", "userID"), time.Now().Add(-time.Second), resourceTypes),
					test.MakeResourceDeleted(commands.NewResourceID("a", "/a"), commands.Status_OK, &commands.Content{}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "3", "userID"), resourceTypes),
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
		Content:       &commands.Content{},
		ResourceTypes: []string{"type1", "type2"},
	},
	ResourceCreatePendings: []*events.ResourceCreatePending{
		{
			ResourceId: &commands.ResourceId{
				DeviceId: "devCreate",
				Href:     "/devCreate",
			},
			ResourceTypes: []string{"type1", "type2"},
		},
	},
	ResourceRetrievePendings: []*events.ResourceRetrievePending{
		{
			ResourceId: &commands.ResourceId{
				DeviceId: "devRetrieve",
				Href:     "/devRetrieve",
			},
			ResourceTypes: []string{"type1", "type2"},
		},
	},
	ResourceUpdatePendings: []*events.ResourceUpdatePending{
		{
			ResourceId: &commands.ResourceId{
				DeviceId: "devUpdate",
				Href:     "/devUpdate",
			},
			ResourceTypes: []string{"type1", "type2"},
		},
	},
	ResourceDeletePendings: []*events.ResourceDeletePending{
		{
			ResourceId: &commands.ResourceId{
				DeviceId: "devDelete",
				Href:     "/devDelete",
			},
			ResourceTypes: []string{"type1", "type2"},
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
	ResourceTypes: []string{"type1", "type2"},
}

func TestResourceStateSnapshotTakenCopyData(t *testing.T) {
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

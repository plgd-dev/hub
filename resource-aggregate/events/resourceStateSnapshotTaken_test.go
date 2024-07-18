package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcgwPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/pb"
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

func TestResourceStateSnapshotHandleCommand(t *testing.T) {
	deviceID := "deviceId"
	correlationID := "correlationID"
	connectionID := "connectionID"
	userID := "userID"
	hubID := "hubID"
	resourceHref := "/a"
	resourceTypes := []string{"type1", "type2"}

	type cmd struct {
		cmd        aggregate.Command
		newVersion uint64
		wantErr    bool
		want       []*grpcgwPb.Event
	}
	tests := []struct {
		name string
		cmds []cmd
	}{
		{
			name: "notify-resource-changed",
			cmds: []cmd{
				{
					cmd: &commands.NotifyResourceChangedRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						Status: commands.Status_OK,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						ResourceTypes: resourceTypes,
					},
					newVersion: 0,
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceChanged{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, "", userID),
							OpenTelemetryCarrier: map[string]string{},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
							Status: commands.Status_OK,
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     0,
							},
							ResourceTypes: resourceTypes,
						}),
					},
				},
			},
		},
		{
			name: "update-resource-not-exists",
			cmds: []cmd{
				{
					cmd: &commands.UpdateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
					},
					wantErr: true,
				},
			},
		},
		{
			name: "update-resource-not-exists-with-flag",
			cmds: []cmd{
				{
					cmd: &commands.UpdateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceUpdatePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
				{
					cmd: &commands.ConfirmResourceUpdateRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceUpdated{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
			},
		},
		{
			name: "update-resource-not-exists-with-flag-and-cancel",
			cmds: []cmd{
				{
					cmd: &commands.UpdateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceUpdatePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
				{
					cmd: &commands.CancelPendingCommandsRequest{
						ResourceId:          commands.NewResourceID(deviceID, resourceHref),
						CorrelationIdFilter: []string{correlationID},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceUpdated{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Status: commands.Status_CANCELED,
						}),
					},
				},
			},
		},
		{
			name: "create-resource-not-exists",
			cmds: []cmd{
				{
					cmd: &commands.CreateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
					},
					wantErr: true,
				},
			},
		},
		{
			name: "create-resource-not-exists-with-flag",
			cmds: []cmd{
				{
					cmd: &commands.CreateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceCreatePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
				{
					cmd: &commands.ConfirmResourceCreateRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceCreated{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
			},
		},
		{
			name: "create-resource-not-exists-with-flag-and-cancel",
			cmds: []cmd{
				{
					cmd: &commands.CreateResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						Content: &commands.Content{
							Data:              []byte("{}"),
							ContentType:       "application/json",
							CoapContentFormat: int32(message.AppJSON),
						},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceCreatePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
							Content: &commands.Content{
								Data:              []byte("{}"),
								ContentType:       "application/json",
								CoapContentFormat: int32(message.AppJSON),
							},
						}),
					},
				},
				{
					cmd: &commands.CancelPendingCommandsRequest{
						ResourceId:          commands.NewResourceID(deviceID, resourceHref),
						CorrelationIdFilter: []string{correlationID},
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceCreated{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Status: commands.Status_CANCELED,
						}),
					},
				},
			},
		},
		{
			name: "delete-resource-not-exists",
			cmds: []cmd{
				{
					cmd: &commands.DeleteResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
					},
					wantErr: true,
				},
			},
		},
		{
			name: "delete-resource-not-exists-with-flag",
			cmds: []cmd{
				{
					cmd: &commands.DeleteResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceDeletePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
						}),
					},
				},
				{
					cmd: &commands.ConfirmResourceDeleteRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						Content: &commands.Content{
							Data:        []byte("{}"),
							ContentType: "application/json",
						},
						CorrelationId: correlationID,
						Status:        commands.Status_METHOD_NOT_ALLOWED,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceDeleted{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Content: &commands.Content{
								Data:        []byte("{}"),
								ContentType: "application/json",
							},
							Status: commands.Status_METHOD_NOT_ALLOWED,
						}),
					},
				},
			},
		},
		{
			name: "delete-resource-not-exists-with-flag-and-cancel",
			cmds: []cmd{
				{
					cmd: &commands.DeleteResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						Force:         true,
						CorrelationId: correlationID,
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceDeletePending{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      0,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     1,
							},
						}),
					},
				},
				{
					cmd: &commands.CancelPendingCommandsRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						CorrelationIdFilter: []string{correlationID},
					},
					want: []*grpcgwPb.Event{
						pb.ToEvent(&events.ResourceDeleted{
							ResourceId:           commands.NewResourceID(deviceID, resourceHref),
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
							EventMetadata: &events.EventMetadata{
								Version:      1,
								Timestamp:    0,
								ConnectionId: connectionID,
								Sequence:     2,
							},
							Status: commands.Status_CANCELED,
						}),
					},
				},
			},
		},
		{
			name: "retrieve-resource-not-exists",
			cmds: []cmd{
				{
					cmd: &commands.RetrieveResourceRequest{
						ResourceId: commands.NewResourceID(deviceID, resourceHref),
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
					},
					wantErr: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewResourceStateSnapshotTakenForCommand(userID, userID, hubID, nil)
			for idx, cmd := range tt.cmds {
				res, err := e.HandleCommand(context.TODO(), cmd.cmd, cmd.newVersion)
				if cmd.wantErr {
					require.Error(t, err, "cmd: %v", idx)
				} else {
					require.NoError(t, err, "cmd: %v", idx)
					var got []*grpcgwPb.Event
					if len(res) > 0 {
						got = make([]*grpcgwPb.Event, 0, len(res))
						for _, e := range res {
							grpcEv := pb.ToEvent(e)
							d1, err := proto.Marshal(grpcEv)
							require.NoError(t, err)
							var v grpcgwPb.Event
							err = proto.Unmarshal(d1, &v)
							require.NoError(t, err)
							got = append(got, &v)
						}
					}
					pb.CmpEvents(t, cmd.want, got)
				}
			}
		})
	}
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

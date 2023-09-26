package events_test

import (
	"context"
	"testing"
	"time"

	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventServicesMetadataSnapshotTaken events.ServicesMetadataSnapshotTaken = events.ServicesMetadataSnapshotTaken{
	ServicesMetadataUpdated: &events.ServicesMetadataUpdated{
		Status: &events.ServicesStatus{
			Online: []*events.ServicesStatus_Status{
				{
					Id:               "0",
					OnlineValidUntil: pkgTime.MaxTime.Unix(),
				},
			},
			Offline: []*events.ServicesStatus_Status{
				{
					Id:               "1",
					OnlineValidUntil: pkgTime.MinTime.Unix(),
				},
			},
		},
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
	EventMetadata: &events.EventMetadata{
		Version:      2,
		Timestamp:    43,
		ConnectionId: "con2",
		Sequence:     2,
	},
}

func TestServicesMetadataSnapshotTakenCopyData(t *testing.T) {
	type args struct {
		event *events.ServicesMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventServicesMetadataSnapshotTaken,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ServicesMetadataSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestServicesMetadataSnapshotTakenCheckInitialized(t *testing.T) {
	type args struct {
		event *events.ServicesMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ServicesMetadataSnapshotTaken{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventServicesMetadataSnapshotTaken,
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

func TestServicesMetadataSnapshotTakenHandle(t *testing.T) {
	type args struct {
		events *iterator
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "online,online-duplicity",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "0",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "0",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "online-1,online-2",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "0",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "1",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "online-1,offline-1",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "0",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Offline: []*events.ServicesStatus_Status{
							{
								Id:               "0",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "snapshot,online",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServicesMetadataSnapshotTaken("a", &events.ServicesMetadataUpdated{
						EventMetadata: events.MakeEventMeta("", 0, 0, "hubID"),
						Status: &events.ServicesStatus{
							Online: []*events.ServicesStatus_Status{
								{
									Id:               "0",
									OnlineValidUntil: pkgTime.MaxTime.Unix(),
								},
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID")),
					test.MakeServicesMetadataUpdated("a", &events.ServicesStatus{
						Online: []*events.ServicesStatus_Status{
							{
								Id:               "1",
								OnlineValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewServicesMetadataSnapshotTaken()
			err := e.Handle(context.TODO(), tt.args.events)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestServicesMetadataSnapshotTakenHandleCommand(t *testing.T) {
	serviceID := "serviceID"
	correlationID := "correlationID"
	userID := "userID"
	hubID := "hubID"

	type cmd struct {
		cmd        aggregate.Command
		newVersion uint64
		wantErr    bool
		want       []eventstore.Event
		sleep      time.Duration
	}
	tests := []struct {
		name string
		cmds []cmd
	}{
		{
			name: "online",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "online,online",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Millisecond * 500,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "online,offline",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID + "1",
									},
								},
								Offline: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "online,offline,online-fail",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID + "1",
									},
								},
								Offline: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 2,
					wantErr:    true,
				},
			},
		},
		{
			name: "online,offline,confirmOffline",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
								Offline: []*events.ServicesStatus_Status{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Status{
							Status: &commands.ServiceStatus{
								Id:         serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServicesMetadataUpdated{
							Status: &events.ServicesStatus{
								Online: []*events.ServicesStatus_Status{
									{
										Id: serviceID + "1",
									},
								},
								Offline: []*events.ServicesStatus_Status{
									{
										Id: serviceID,
									},
								},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					cmd: &events.ConfirmOfflineServicesRequest{
						Status: []*events.ServicesStatus_Status{
							{
								Id: serviceID,
							},
						},
					},
					newVersion: 2,
					want: []eventstore.Event{
						&events.ServicesMetadataSnapshotTaken{
							ServicesMetadataUpdated: &events.ServicesMetadataUpdated{
								Status: &events.ServicesStatus{
									Online: []*events.ServicesStatus_Status{
										{
											Id: serviceID + "1",
										},
									},
									Offline: []*events.ServicesStatus_Status{},
								},
								AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
								OpenTelemetryCarrier: map[string]string{},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewServicesMetadataSnapshotTakenForCommand(userID, userID, hubID)
			for idx, cmd := range tt.cmds {
				if cmd.sleep > 0 {
					time.Sleep(cmd.sleep)
				}
				got, err := e.HandleCommand(context.TODO(), cmd.cmd, cmd.newVersion)
				if cmd.wantErr {
					require.Error(t, err, "cmd: %v", idx)
					return
				}
				require.NoError(t, err, "cmd: %v", idx)
				pb.CmpServicesEvents(t, cmd.want, got)
			}
		})
	}
}

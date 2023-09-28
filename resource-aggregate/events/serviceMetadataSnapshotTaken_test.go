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

var testEventServiceMetadataSnapshotTaken events.ServiceMetadataSnapshotTaken = events.ServiceMetadataSnapshotTaken{
	ServiceMetadataUpdated: &events.ServiceMetadataUpdated{
		ServicesHeartbeat: &events.ServicesHeartbeat{
			Valid: []*events.ServicesHeartbeat_Heartbeat{
				{
					ServiceId:  "0",
					ValidUntil: pkgTime.MaxTime.Unix(),
				},
			},
			Expired: []*events.ServicesHeartbeat_Heartbeat{
				{
					ServiceId:  "1",
					ValidUntil: pkgTime.MinTime.Unix(),
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

func TestServiceMetadataSnapshotTakenCopyData(t *testing.T) {
	type args struct {
		event *events.ServiceMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventServiceMetadataSnapshotTaken,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ServiceMetadataSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestServiceMetadataSnapshotTakenCheckInitialized(t *testing.T) {
	type args struct {
		event *events.ServiceMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ServiceMetadataSnapshotTaken{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventServiceMetadataSnapshotTaken,
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

func TestServiceMetadataSnapshotTakenHandle(t *testing.T) {
	type args struct {
		events *iterator
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid,valid-duplicity",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "0",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "0",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "valid-1,valid-2",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "0",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "1",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "valid-1,expired-1",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "0",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Expired: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "0",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
		{
			name: "snapshot,valid",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeServiceMetadataSnapshotTaken("a", &events.ServiceMetadataUpdated{
						EventMetadata: events.MakeEventMeta("", 0, 0, "hubID"),
						ServicesHeartbeat: &events.ServicesHeartbeat{
							Valid: []*events.ServicesHeartbeat_Heartbeat{
								{
									ServiceId:  "0",
									ValidUntil: pkgTime.MaxTime.Unix(),
								},
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID")),
					test.MakeServiceMetadataUpdated("a", &events.ServicesHeartbeat{
						Valid: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId:  "1",
								ValidUntil: pkgTime.MaxTime.Unix(),
							},
						},
					}, events.MakeEventMeta("", 0, 0, "hubID"), commands.NewAuditContext("userID", "0", "userID")),
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewServiceMetadataSnapshotTaken()
			err := e.Handle(context.TODO(), tt.args.events)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestServiceMetadataSnapshotTakenHandleCommand(t *testing.T) {
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
			name: "valid",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "valid,valid",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Millisecond * 500,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
			},
		},
		{
			name: "valid,expired",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID + "1",
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
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
			name: "valid,expired,valid-fail",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID + "1",
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
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
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
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
			name: "valid,expired,confirmExpired",
			cmds: []cmd{
				{
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID,
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 0,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					sleep: time.Second,
					cmd: &commands.UpdateServiceMetadataRequest{
						Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
							Heartbeat: &commands.ServiceHeartbeat{
								ServiceId:  serviceID + "1",
								TimeToLive: time.Second.Nanoseconds(),
							},
						},
					},
					newVersion: 1,
					want: []eventstore.Event{
						&events.ServiceMetadataUpdated{
							ServicesHeartbeat: &events.ServicesHeartbeat{
								Valid: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID + "1",
									},
								},
								Expired: []*events.ServicesHeartbeat_Heartbeat{
									{
										ServiceId: serviceID,
									},
								},
							},
							AuditContext:         commands.NewAuditContext(userID, correlationID, userID),
							OpenTelemetryCarrier: map[string]string{},
						},
					},
				},
				{
					cmd: &events.ConfirmExpiredServicesRequest{
						Heartbeat: []*events.ServicesHeartbeat_Heartbeat{
							{
								ServiceId: serviceID,
							},
						},
					},
					newVersion: 2,
					want: []eventstore.Event{
						&events.ServiceMetadataSnapshotTaken{
							ServiceMetadataUpdated: &events.ServiceMetadataUpdated{
								ServicesHeartbeat: &events.ServicesHeartbeat{
									Valid: []*events.ServicesHeartbeat_Heartbeat{
										{
											ServiceId: serviceID + "1",
										},
									},
									Expired: []*events.ServicesHeartbeat_Heartbeat{},
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
			e := events.NewServiceMetadataSnapshotTakenForCommand(userID, userID, hubID)
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

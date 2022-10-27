package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	grpcgwPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testEventDeviceMetadataSnapshotTaken events.DeviceMetadataSnapshotTaken = events.DeviceMetadataSnapshotTaken{
	DeviceId: "dev1",
	DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
		DeviceId: "dev1",
		Connection: &commands.Connection{
			Status:           commands.Connection_ONLINE,
			OnlineValidUntil: 12345,
			Id:               "con1",
		},
		TwinEnabled: true,
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
	UpdatePendings: []*events.DeviceMetadataUpdatePending{
		{
			DeviceId: "dev1",
		},
	},
	EventMetadata: &events.EventMetadata{
		Version:      2,
		Timestamp:    43,
		ConnectionId: "con2",
		Sequence:     2,
	},
}

func TestDeviceMetadataSnapshotTakenCopyData(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &testEventDeviceMetadataSnapshotTaken,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.DeviceMetadataSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

func TestDeviceMetadataSnapshotTakenCheckInitialized(t *testing.T) {
	type args struct {
		event *events.DeviceMetadataSnapshotTaken
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.DeviceMetadataSnapshotTaken{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventDeviceMetadataSnapshotTaken,
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

func TestDeviceMetadataSnapshotTakenHandle(t *testing.T) {
	type args struct {
		events *iterator
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "twinSyncPending, updated",
			args: args{
				events: newIterator([]eventstore.EventUnmarshaler{
					test.MakeDeviceMetadataUpdatePending("a", &events.DeviceMetadataUpdatePending_TwinEnabled{
						TwinEnabled: true,
					}, events.MakeEventMeta("", 0, 0), commands.NewAuditContext("userID", "0"), time.Now().Add(-time.Second)),
					test.MakeDeviceMetadataUpdated("a", &commands.Connection{Id: "123"}, true, events.MakeEventMeta("", 0, 0), commands.NewAuditContext("userID", "0"), false),
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewDeviceMetadataSnapshotTaken()
			err := e.Handle(context.TODO(), tt.args.events)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeviceMetadataSnapshotTakenHandleCommand(t *testing.T) {
	deviceID := "deviceId"
	correlationID := "correlationID"
	connectionID := "connectionID"
	userID := "userID"
	connectedAt := int64(1235)
	jwtWithSubUserID := config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})

	type args struct {
		ctx        context.Context
		cmd        aggregate.Command
		newVersion uint64
	}
	type preCmds struct {
		ctx        context.Context
		cmd        aggregate.Command
		newVersion uint64
		wantErr    bool
	}
	tests := []struct {
		name    string
		preCmds []preCmds
		args    args
		want    []*grpcgwPb.Event
		wantErr bool
	}{
		{
			name: "online,online,offline",
			preCmds: []preCmds{
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						TimeToLive:    0,
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status:      commands.Connection_ONLINE,
								ConnectedAt: connectedAt,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						TimeToLive:    0,
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status: commands.Connection_ONLINE,
							},
						},
					},
					newVersion: 0,
				},
			},
			args: args{
				newVersion: 1,
				ctx:        grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				cmd: &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: connectionID,
						Sequence:     0,
					},
					TimeToLive:    0,
					CorrelationId: correlationID,
					Update: &commands.UpdateDeviceMetadataRequest_Connection{
						Connection: &commands.Connection{
							Status: commands.Connection_OFFLINE,
						},
					},
				},
			},
			want: []*grpcgwPb.Event{
				pb.ToEvent(&events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:      commands.Connection_OFFLINE,
						ConnectedAt: connectedAt,
					},
					TwinEnabled:          true,
					TwinSynchronization:  &commands.TwinSynchronization{},
					AuditContext:         commands.NewAuditContext(userID, correlationID),
					OpenTelemetryCarrier: map[string]string{},
				}),
			},
		},
		{
			name: "online-old-connection,online,offline-old-connection",
			preCmds: []preCmds{
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						TimeToLive:    0,
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status:      commands.Connection_ONLINE,
								ConnectedAt: connectedAt,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID + "1",
							Sequence:     0,
						},
						TimeToLive:    0,
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status: commands.Connection_ONLINE,
							},
						},
					},
					newVersion: 1,
				},
			},
			args: args{
				newVersion: 2,
				ctx:        grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				cmd: &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: connectionID,
						Sequence:     0,
					},
					TimeToLive:    0,
					CorrelationId: correlationID,
					Update: &commands.UpdateDeviceMetadataRequest_Connection{
						Connection: &commands.Connection{
							Status: commands.Connection_OFFLINE,
						},
					},
				},
			},
		},
		{
			name: "empty ConnectionStatus.ConnectionId",
			args: args{
				newVersion: 1,
				ctx:        grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				cmd: &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					CommandMetadata: &commands.CommandMetadata{
						Sequence: 0,
					},
					TimeToLive:    0,
					CorrelationId: correlationID,
					Update: &commands.UpdateDeviceMetadataRequest_Connection{
						Connection: &commands.Connection{
							Status: commands.Connection_ONLINE,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "online,twin-sync-started,twin-sync-started,twin-sync-finished,twin-sync-finished",
			preCmds: []preCmds{
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status:      commands.Connection_ONLINE,
								ConnectedAt: connectedAt,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:     commands.TwinSynchronization_STARTED,
								StartedAt: 1,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:     commands.TwinSynchronization_STARTED,
								StartedAt: 2,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     3,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:      commands.TwinSynchronization_FINISHED,
								FinishedAt: 3,
							},
						},
					},
					newVersion: 0,
				},
			},
			args: args{
				newVersion: 1,
				ctx:        grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				cmd: &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: connectionID,
						Sequence:     4,
					},
					CorrelationId: correlationID,
					Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
						TwinSynchronization: &commands.TwinSynchronization{
							State:      commands.TwinSynchronization_FINISHED,
							FinishedAt: 4,
						},
					},
				},
			},
			want: []*grpcgwPb.Event{
				pb.ToEvent(&events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:      commands.Connection_ONLINE,
						ConnectedAt: connectedAt,
					},
					TwinEnabled: true,
					TwinSynchronization: &commands.TwinSynchronization{
						StartedAt:  1,
						FinishedAt: 4,
						State:      commands.TwinSynchronization_FINISHED,
					},
					AuditContext:         commands.NewAuditContext(userID, correlationID),
					OpenTelemetryCarrier: map[string]string{},
				}),
			},
		},
		{
			name: "online-old,twin-sync-started-old,online,twin-sync-started,twin-sync-finished-old,twin-sync-finished",
			preCmds: []preCmds{
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID + "1",
							Sequence:     0,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status:      commands.Connection_ONLINE,
								ConnectedAt: connectedAt,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID + "1",
							Sequence:     1,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:     commands.TwinSynchronization_STARTED,
								StartedAt: 1,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     0,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_Connection{
							Connection: &commands.Connection{
								Status: commands.Connection_ONLINE,

								ConnectedAt: connectedAt,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     1,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:     commands.TwinSynchronization_STARTED,
								StartedAt: 1,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID,
							Sequence:     2,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:     commands.TwinSynchronization_STARTED,
								StartedAt: 2,
							},
						},
					},
					newVersion: 0,
				},
				{
					ctx: grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
					cmd: &commands.UpdateDeviceMetadataRequest{
						DeviceId: deviceID,
						CommandMetadata: &commands.CommandMetadata{
							ConnectionId: connectionID + "1",
							Sequence:     2,
						},
						CorrelationId: correlationID,
						Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
							TwinSynchronization: &commands.TwinSynchronization{
								State:      commands.TwinSynchronization_FINISHED,
								FinishedAt: 3,
							},
						},
					},
					newVersion: 0,
					wantErr:    true,
				},
			},
			args: args{
				newVersion: 1,
				ctx:        grpc.CtxWithIncomingToken(context.Background(), jwtWithSubUserID),
				cmd: &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: connectionID,
						Sequence:     3,
					},
					CorrelationId: correlationID,
					Update: &commands.UpdateDeviceMetadataRequest_TwinSynchronization{
						TwinSynchronization: &commands.TwinSynchronization{
							State:      commands.TwinSynchronization_FINISHED,
							FinishedAt: 3,
						},
					},
				},
			},
			want: []*grpcgwPb.Event{
				pb.ToEvent(&events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status:      commands.Connection_ONLINE,
						ConnectedAt: connectedAt,
					},
					TwinEnabled: true,
					TwinSynchronization: &commands.TwinSynchronization{
						StartedAt:  1,
						FinishedAt: 3,
						State:      commands.TwinSynchronization_FINISHED,
					},
					AuditContext:         commands.NewAuditContext(userID, correlationID),
					OpenTelemetryCarrier: map[string]string{},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := events.NewDeviceMetadataSnapshotTaken()
			for idx, preCmd := range tt.preCmds {
				_, err := e.HandleCommand(preCmd.ctx, preCmd.cmd, preCmd.newVersion)
				if preCmd.wantErr {
					require.Error(t, err, "precmd: %v", idx)
				} else {
					require.NoError(t, err, "precmd: %v", idx)
				}
			}
			res, err := e.HandleCommand(tt.args.ctx, tt.args.cmd, tt.args.newVersion)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var got []*grpcgwPb.Event
			if len(res) > 0 {
				got = make([]*grpcgwPb.Event, 0, len(res))
				for _, e := range res {
					got = append(got, pb.ToEvent(e))
				}
			}
			pb.CmpEvents(t, tt.want, got)
		})
	}
}

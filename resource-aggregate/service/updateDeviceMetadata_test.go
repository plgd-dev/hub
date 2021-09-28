package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	raTest "github.com/plgd-dev/cloud/resource-aggregate/test"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newConnectionStatus(v commands.ConnectionStatus_Status) *commands.ConnectionStatus_Status {
	return &v
}

func TestAggregateHandle_UpdateDeviceMetadata(t *testing.T) {
	type args struct {
		request *commands.UpdateDeviceMetadataRequest
		userID  string
	}

	test := []struct {
		name    string
		args    args
		want    codes.Code
		wantErr bool
	}{

		{
			name: "set online",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest("dev0", "", newConnectionStatus(commands.ConnectionStatus_ONLINE), commands.ShadowSynchronization_UNSET, time.Hour),
				userID:  "user0",
			},
			want: codes.OK,
		},
		{
			name: "set shadowSynchronizationDisabled",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest("dev0", "", nil, commands.ShadowSynchronization_DISABLED, time.Hour),
				userID:  "user0",
			},
			want: codes.OK,
		},
		{
			name: "invalid valid until",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest("dev0", "", nil, commands.ShadowSynchronization_DISABLED, -time.Hour),
				userID:  "user0",
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
		{
			name: "invalid update commands",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest("dev0", "", nil, commands.ShadowSynchronization_UNSET, time.Hour),
				userID:  "user0",
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log)

	fmt.Printf("%v\n", cfg.String())

	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	assert.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			ctx := kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			events, err := ag.UpdateDeviceMetadata(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
			} else {
				require.NoError(t, err)
				err = service.PublishEvents(ctx, publisher, tt.args.request.GetDeviceId(), ag.ResourceID(), events)
				assert.NoError(t, err)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_UpdateDeviceMetadata(t *testing.T) {
	deviceID := "dev0"
	user0 := "user0"
	type args struct {
		request *commands.UpdateDeviceMetadataRequest
		sleep   time.Duration
	}
	test := []struct {
		name      string
		args      args
		want      *commands.UpdateDeviceMetadataResponse
		wantError bool
	}{
		{
			name: "set online",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest(deviceID, "", newConnectionStatus(commands.ConnectionStatus_ONLINE), commands.ShadowSynchronization_UNSET, time.Hour),
			},
			want: &commands.UpdateDeviceMetadataResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest(deviceID, "", newConnectionStatus(commands.ConnectionStatus_ONLINE), commands.ShadowSynchronization_UNSET, time.Hour),
			},
			want: &commands.UpdateDeviceMetadataResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
		},
		{
			name: "set shadowSynchronizationDisabled - with expiration",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, commands.ShadowSynchronization_DISABLED, time.Millisecond*250),
				sleep:   time.Millisecond * 500,
			},
			want: &commands.UpdateDeviceMetadataResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Millisecond * 250)),
			},
		},
		{
			name: "set shadowSynchronizationDisabled",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, commands.ShadowSynchronization_DISABLED, time.Hour),
			},
			want: &commands.UpdateDeviceMetadataResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
			},
		},
		{
			name: "invalid",
			args: args{
				request: testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, commands.ShadowSynchronization_UNSET, time.Hour),
			},
			wantError: true,
		},
		{
			name: "empty",
			args: args{
				request: &commands.UpdateDeviceMetadataRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.UpdateDeviceMetadata(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want != nil {
				assert.Equal(t, tt.want.AuditContext, response.AuditContext)
			}
			if tt.want.GetValidUntil() == 0 {
				assert.Equal(t, tt.want.ValidUntil, response.GetValidUntil())
			} else {
				assert.Less(t, tt.want.ValidUntil, response.GetValidUntil())
			}
			time.Sleep(tt.args.sleep)
		}
		t.Run(tt.name, tfunc)
	}
}

func testMakeUpdateDeviceMetadataRequest(deviceID, correlationID string, online *commands.ConnectionStatus_Status, shadowSynchronization commands.ShadowSynchronization, timeToLive time.Duration) *commands.UpdateDeviceMetadataRequest {
	r := commands.UpdateDeviceMetadataRequest{
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
		TimeToLive:    int64(timeToLive),
		CorrelationId: correlationID,
	}
	if online != nil {
		r.Update = &commands.UpdateDeviceMetadataRequest_Status{
			Status: &commands.ConnectionStatus{
				Value: *online,
			},
		}
	}
	if shadowSynchronization != commands.ShadowSynchronization_UNSET {
		r.Update = &commands.UpdateDeviceMetadataRequest_ShadowSynchronization{
			ShadowSynchronization: shadowSynchronization,
		}
	}
	return &r
}

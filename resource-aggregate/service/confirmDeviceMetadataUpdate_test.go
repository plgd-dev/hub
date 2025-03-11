package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAggregateHandleConfirmDeviceMetadataUpdate(t *testing.T) {
	deviceID := dev1
	const userID = "user1"
	const owner = userID
	type args struct {
		request *commands.ConfirmDeviceMetadataUpdateRequest
		userID  string
	}

	test := []struct {
		name    string
		args    args
		want    codes.Code
		wantErr bool
	}{
		{
			name: "set twinSynchronizationDisabled",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, newTwinEnabled(true)),
				userID:  userID,
			},
			want: codes.OK,
		},
		{
			name: "set twinSynchronizationDisabled duplicit with same correlationID",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, newTwinEnabled(false)),
				userID:  userID,
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
		{
			name: "invalid update commands",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, nil),
				userID:  userID,
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.StatusHref), eventstore, service.NewDeviceMetadataFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(ctx, testMakeUpdateDeviceMetadataRequest(deviceID, "a", newConnectionStatus(commands.Connection_ONLINE), nil, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(ctx, testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, newTwinEnabled(false), time.Hour))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), eventstore, service.NewDeviceMetadataFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			events, err := ag.ConfirmDeviceMetadataUpdate(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
				return
			}
			require.NoError(t, err)
			service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events, logger)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmDeviceMetadataUpdate(t *testing.T) {
	deviceID := dev0
	const userID = "user0"
	type args struct {
		request *commands.ConfirmDeviceMetadataUpdateRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.ConfirmDeviceMetadataUpdateResponse
		wantError bool
	}{
		{
			name: "set twinSynchronizationDisabled",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, newTwinEnabled(false)),
			},
			want: &commands.ConfirmDeviceMetadataUpdateResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, newTwinEnabled(false)),
			},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, nil),
			},
			wantError: true,
		},
		{
			name: "empty",
			args: args{
				request: &commands.ConfirmDeviceMetadataUpdateRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	_, err = requestHandler.UpdateDeviceMetadata(ctx, testMakeUpdateDeviceMetadataRequest(deviceID, "", newConnectionStatus(commands.Connection_ONLINE), nil, time.Hour))
	require.NoError(t, err)
	_, err = requestHandler.UpdateDeviceMetadata(ctx, testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, newTwinEnabled(false), time.Hour))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.ConfirmDeviceMetadataUpdate(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want != nil {
				assert.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testMakeConfirmDeviceMetadataUpdateRequest(deviceID string, twinEnabled *bool) *commands.ConfirmDeviceMetadataUpdateRequest {
	r := commands.ConfirmDeviceMetadataUpdateRequest{
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	if twinEnabled != nil {
		r.Confirm = &commands.ConfirmDeviceMetadataUpdateRequest_TwinEnabled{
			TwinEnabled: *twinEnabled,
		}
	}
	return &r
}

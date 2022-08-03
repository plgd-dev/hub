package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
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
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAggregateHandleConfirmDeviceMetadataUpdate(t *testing.T) {
	const deviceID = "dev1"
	const userID = "user1"
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
			name: "set shadowSynchronizationDisabled",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_ENABLED),
				userID:  userID,
			},
			want: codes.OK,
		},
		{
			name: "set shadowSynchronizationDisabled duplicit with same correlationID",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_DISABLED),
				userID:  userID,
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
		{
			name: "invalid update commands",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_UNSET),
				userID:  userID,
			},
			want:    codes.InvalidArgument,
			wantErr: true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		err := fileWatcher.Close()
		require.NoError(t, err)
	}()

	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, commands.ShadowSynchronization_DISABLED, time.Hour))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			ctx := kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			events, err := ag.ConfirmDeviceMetadataUpdate(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
				return
			}
			require.NoError(t, err)
			err = service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events)
			assert.NoError(t, err)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmDeviceMetadataUpdate(t *testing.T) {
	const deviceID = "dev0"
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
			name: "set shadowSynchronizationDisabled",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_DISABLED),
			},
			want: &commands.ConfirmDeviceMetadataUpdateResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_DISABLED),
			},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: testMakeConfirmDeviceMetadataUpdateRequest(deviceID, commands.ShadowSynchronization_UNSET),
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
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		err := fileWatcher.Close()
		require.NoError(t, err)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)

	_, err = requestHandler.UpdateDeviceMetadata(ctx, testMakeUpdateDeviceMetadataRequest(deviceID, "", nil, commands.ShadowSynchronization_DISABLED, time.Hour))
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
				assert.Equal(t, tt.want.AuditContext, response.AuditContext)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testMakeConfirmDeviceMetadataUpdateRequest(deviceID string, shadowSynchronization commands.ShadowSynchronization) *commands.ConfirmDeviceMetadataUpdateRequest {
	r := commands.ConfirmDeviceMetadataUpdateRequest{
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	if shadowSynchronization != commands.ShadowSynchronization_UNSET {
		r.Confirm = &commands.ConfirmDeviceMetadataUpdateRequest_ShadowSynchronization{
			ShadowSynchronization: shadowSynchronization,
		}
	}
	return &r
}

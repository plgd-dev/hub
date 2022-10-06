package service_test

import (
	"context"
	"testing"

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

func TestAggregateHandleCancelPendingMetadataUpdates(t *testing.T) {
	const deviceID = "dev0"
	const userID = "user0"
	const correlationID0 = "0"
	const correlationID1 = "1"
	const correlationID2 = "2"
	type args struct {
		request *commands.CancelPendingMetadataUpdatesRequest
		userID  string
	}

	test := []struct {
		name     string
		args     args
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name: "cancel one update",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, []string{correlationID0}),
				userID:  userID,
			},
			wantCode: codes.OK,
		},
		{
			name: "cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
			},
			wantCode: codes.OK,
		},
		{
			name: "duplicit cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
			},
			wantCode: codes.NotFound,
			wantErr:  true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
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
		errC := eventstore.Close(ctx)
		assert.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, commands.ShadowSynchronization_ENABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			ctx := kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			events, err := ag.CancelPendingMetadataUpdates(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
				return
			}
			require.NoError(t, err)
			err = service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events)
			assert.NoError(t, err)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerCancelPendingMetadataUpdates(t *testing.T) {
	const deviceID = "dev1"
	const userID = "user1"
	const correlationID0 = "0"
	const correlationID1 = "1"
	const correlationID2 = "2"
	type args struct {
		request *commands.CancelPendingMetadataUpdatesRequest
		userID  string
	}

	test := []struct {
		name     string
		args     args
		want     *commands.CancelPendingMetadataUpdatesResponse
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name: "cancel one update",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, []string{correlationID0}),
				userID:  userID,
			},
			want: &commands.CancelPendingMetadataUpdatesResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
				},
				CorrelationIds: []string{correlationID0},
			},
			wantCode: codes.OK,
		},
		{
			name: "cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
			},
			want: &commands.CancelPendingMetadataUpdatesResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
				},
				CorrelationIds: []string{correlationID1, correlationID2},
			},
			wantCode: codes.OK,
		},
		{
			name: "duplicit cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
			},
			wantCode: codes.NotFound,
			wantErr:  true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
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
		errC := eventstore.Close(ctx)
		assert.NoError(t, errC)
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
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, commands.ShadowSynchronization_ENABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	})), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ctx := kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			want, err := requestHandler.CancelPendingMetadataUpdates(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, want)
		}
		t.Run(tt.name, tfunc)
	}
}

func testMakeCancelPendingMetadataUpdatesRequest(deviceID string, correlationIdFilter []string) *commands.CancelPendingMetadataUpdatesRequest {
	r := commands.CancelPendingMetadataUpdatesRequest{
		DeviceId:            deviceID,
		CorrelationIdFilter: correlationIdFilter,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

package service_test

import (
	"context"
	"testing"

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

func TestAggregateHandleCancelPendingMetadataUpdates(t *testing.T) {
	deviceID := dev0
	const userID = "user0"
	const owner = "owner0"
	const correlationID0 = "0"
	const correlationID1 = "1"
	const correlationID2 = "2"
	type args struct {
		request *commands.CancelPendingMetadataUpdatesRequest
		userID  string
		owner   string
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
				owner:   owner,
			},
			wantCode: codes.OK,
		},
		{
			name: "cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
				owner:   owner,
			},
			wantCode: codes.OK,
		},
		{
			name: "duplicit cancel all updates",
			args: args{
				request: testMakeCancelPendingMetadataUpdatesRequest(deviceID, nil),
				userID:  userID,
				owner:   owner,
			},
			wantCode: codes.NotFound,
			wantErr:  true,
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
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, newConnectionStatus(commands.Connection_ONLINE), nil, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, newTwinEnabled(false), 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, newTwinEnabled(true), 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, newTwinEnabled(false), 0))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), eventstore, service.NewDeviceMetadataFactoryModel(tt.args.userID, tt.args.owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			events, err := ag.CancelPendingMetadataUpdates(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
				return
			}
			require.NoError(t, err)
			service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events, logger)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerCancelPendingMetadataUpdates(t *testing.T) {
	deviceID := dev1
	const userID = "user1"
	const owner = userID
	const correlationID0 = "0"
	const correlationID1 = "1"
	const correlationID2 = "2"
	type args struct {
		request *commands.CancelPendingMetadataUpdatesRequest
		userID  string
		owner   string
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
				owner:   owner,
			},
			want: &commands.CancelPendingMetadataUpdatesResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  owner,
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
				owner:   owner,
			},
			want: &commands.CancelPendingMetadataUpdatesResponse{
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  owner,
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
				owner:   owner,
			},
			wantCode: codes.NotFound,
			wantErr:  true,
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
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, newConnectionStatus(commands.Connection_ONLINE), nil, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, newTwinEnabled(false), 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, newTwinEnabled(true), 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(context.Background(), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, newTwinEnabled(false), 0))
	require.NoError(t, err)

	serviceHeartbeat := service.NewServiceHeartbeat(cfg, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			cpmuCtx := kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			want, err := requestHandler.CancelPendingMetadataUpdates(cpmuCtx, tt.args.request)
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

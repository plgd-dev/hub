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

func TestRequestHandlerCancelPendingCommands(t *testing.T) {
	deviceID := dev0
	const resID0 = "res0"
	const resID1 = "res1"
	const userID = "user0"
	const correlationID0 = "0"
	const correlationID1 = "1"
	const correlationID2 = "2"
	const correlationID3 = "3"

	testMakeCancelPendingCommandsRequest := func(deviceID string, href string, correlationIdFilter []string) *commands.CancelPendingCommandsRequest {
		r := commands.CancelPendingCommandsRequest{
			ResourceId:          commands.NewResourceID(deviceID, href),
			CorrelationIdFilter: correlationIdFilter,
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: uuid.Must(uuid.NewRandom()).String(),
				Sequence:     0,
			},
		}
		return &r
	}

	type args struct {
		request *commands.CancelPendingCommandsRequest
	}
	tests := []struct {
		name     string
		args     args
		want     *commands.CancelPendingCommandsResponse
		wantCode codes.Code
		wantErr  bool
	}{
		{
			name: "cancel one update",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID0, []string{correlationID0}),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID0},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel 2 updates",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID1, []string{correlationID0, correlationID1}),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID0, correlationID1},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel one retrieve",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID0, []string{correlationID1}),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID1},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel one create",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID0, []string{correlationID2}),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID2},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel one delete",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID0, []string{correlationID3}),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID3},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel all commands",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID1, nil),
			},
			want: &commands.CancelPendingCommandsResponse{
				CorrelationIds: []string{correlationID2, correlationID3},
				AuditContext: &commands.AuditContext{
					UserId: userID,
					Owner:  userID,
				},
			},
		},
		{
			name: "cancel all commands",
			args: args{
				request: testMakeCancelPendingCommandsRequest(deviceID, resID1, nil),
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
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

	ag0, err := service.NewAggregate(commands.NewResourceID(deviceID, resID0), eventstore, service.NewResourceStateFactoryModel(userID, userID, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	ag1, err := service.NewAggregate(commands.NewResourceID(deviceID, resID1), eventstore, service.NewResourceStateFactoryModel(userID, userID, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag0.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID0, 0, []byte(resID0)...))
	require.NoError(t, err)
	_, err = ag1.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID1, 0, []byte(resID1)...))
	require.NoError(t, err)
	_, err = ag0.UpdateResource(ctx, testMakeUpdateResourceRequest(deviceID, resID0, "", correlationID0, 0))
	require.NoError(t, err)
	_, err = ag1.UpdateResource(ctx, testMakeUpdateResourceRequest(deviceID, resID1, "", correlationID0, 0))
	require.NoError(t, err)
	_, err = ag0.RetrieveResource(ctx, testMakeRetrieveResourceRequest(deviceID, resID0, correlationID1, 0))
	require.NoError(t, err)
	_, err = ag1.RetrieveResource(ctx, testMakeRetrieveResourceRequest(deviceID, resID1, correlationID1, 0))
	require.NoError(t, err)
	_, err = ag0.CreateResource(ctx, testMakeCreateResourceRequest(deviceID, resID0, correlationID2, 0))
	require.NoError(t, err)
	_, err = ag1.CreateResource(ctx, testMakeCreateResourceRequest(deviceID, resID1, correlationID2, 0))
	require.NoError(t, err)
	_, err = ag0.DeleteResource(ctx, testMakeDeleteResourceRequest(deviceID, resID0, correlationID3, 0))
	require.NoError(t, err)
	_, err = ag1.DeleteResource(ctx, testMakeDeleteResourceRequest(deviceID, resID1, correlationID3, 0))
	require.NoError(t, err)

	serviceHeartbeat := service.NewServiceHeartbeat(cfg, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := requestHandler.CancelPendingCommands(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, want)
		})
	}
}

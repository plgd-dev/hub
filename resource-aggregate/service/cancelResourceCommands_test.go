package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/resource-aggregate/test"
	"github.com/plgd-dev/hub/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_CancelPendingCommands(t *testing.T) {
	deviceID := "dev0"
	resID0 := "res0"
	resID1 := "res1"
	userID := "user0"
	correlationID0 := "0"
	correlationID1 := "1"
	correlationID2 := "2"
	correlationID3 := "3"

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

	ag0, err := service.NewAggregate(commands.NewResourceID(deviceID, resID0), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	ag1, err := service.NewAggregate(commands.NewResourceID(deviceID, resID1), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
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

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := requestHandler.CancelPendingCommands(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, want)
			}
		})
	}
}

func testMakeCancelPendingCommandsRequest(deviceID string, href string, correlationIdFilter []string) *commands.CancelPendingCommandsRequest {
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

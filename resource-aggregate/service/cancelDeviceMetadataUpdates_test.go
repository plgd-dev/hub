package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	raTest "github.com/plgd-dev/cloud/resource-aggregate/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAggregateHandle_CancelPendingMetadataUpdates(t *testing.T) {
	deviceID := "dev0"
	userID := "user0"
	correlationID0 := "0"
	correlationID1 := "1"
	correlationID2 := "2"
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
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
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
	publisher, err := publisher.New(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer publisher.Close()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, commands.ShadowSynchronization_ENABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			events, err := ag.CancelPendingMetadataUpdates(kitNetGrpc.CtxWithIncomingOwner(ctx, tt.args.userID), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
			} else {
				require.NoError(t, err)
				err = service.PublishEvents(ctx, publisher, tt.args.request.GetDeviceId(), ag.ResourceID(), events)
				assert.NoError(t, err)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_CancelPendingMetadataUpdates(t *testing.T) {
	deviceID := "dev0"
	userID := "user0"
	correlationID0 := "0"
	correlationID1 := "1"
	correlationID2 := "2"
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
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
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
	publisher, err := publisher.New(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer publisher.Close()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.StatusHref), 10, eventstore, service.DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID0, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID1, nil, commands.ShadowSynchronization_ENABLED, 0))
	require.NoError(t, err)
	_, err = ag.UpdateDeviceMetadata(kitNetGrpc.CtxWithIncomingOwner(ctx, userID), testMakeUpdateDeviceMetadataRequest(deviceID, correlationID2, nil, commands.ShadowSynchronization_DISABLED, 0))
	require.NoError(t, err)

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetUserDevices)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			want, err := requestHandler.CancelPendingMetadataUpdates(kitNetGrpc.CtxWithIncomingOwner(ctx, tt.args.userID), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantCode, s.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, want)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testMakeCancelPendingMetadataUpdatesRequest(deviceID string, correlationIdFilter []string) *commands.CancelPendingMetadataUpdatesRequest {
	r := commands.CancelPendingMetadataUpdatesRequest{
		DeviceId:            deviceID,
		CorrelationIdFilter: correlationIdFilter,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

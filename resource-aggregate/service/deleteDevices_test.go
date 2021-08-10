package service_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	raTest "github.com/plgd-dev/cloud/resource-aggregate/test"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_DeleteDevices(t *testing.T) {
	const deviceID = "dev0"
	const user0 = "user0"

	config := raTest.MakeConfig(t)
	fmt.Printf("%v\n", config.String())
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		require.NoError(t, err)
		_ = eventstore.Close(ctx)
	}()
	publisher, err := publisher.New(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer publisher.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)

	type args struct {
		req   *commands.DeleteDevicesRequest
		owner string
	}
	tests := []struct {
		name      string
		args      args
		want      *commands.DeleteDevicesResponse
		wantError bool
	}{
		{
			name: "unauthorized user",
			args: args{
				req: &commands.DeleteDevicesRequest{
					DeviceIds: []string{"deviceID"},
				},
				owner: testUnauthorizedUser,
			},
			wantError: true,
		},
		{
			name: "non-owned device",
			args: args{
				req: &commands.DeleteDevicesRequest{
					DeviceIds: []string{"testDev0"},
				},
				owner: user0,
			},
			want: &commands.DeleteDevicesResponse{
				DeviceIds: nil,
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
			wantError: false,
		},
		{
			name: "owned and not-owned devices",
			args: args{
				req: &commands.DeleteDevicesRequest{
					DeviceIds: []string{"testDev0", deviceID},
				},
				owner: user0,
			},
			want: &commands.DeleteDevicesResponse{
				DeviceIds: []string{deviceID},
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
			wantError: false,
		},
		{
			name: "all owned devices",
			args: args{
				req: &commands.DeleteDevicesRequest{
					DeviceIds: []string{},
				},
				owner: user0,
			},
			want: &commands.DeleteDevicesResponse{
				DeviceIds: testUserDevices,
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteDevicesCtx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), tt.args.owner)
			response, err := requestHandler.DeleteDevices(deleteDevicesCtx, tt.args.req)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want != nil {
				sort.Strings(tt.want.DeviceIds)
				sort.Strings(response.DeviceIds)
				require.Equal(t, tt.want.DeviceIds, response.DeviceIds)
				require.Equal(t, tt.want.AuditContext, response.AuditContext)
			}
		})
	}
}

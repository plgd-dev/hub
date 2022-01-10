package service_test

import (
	"context"
	"sort"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestRequestHandler_DeleteDevices(t *testing.T) {
	const deviceID = "dev0"
	const user0 = "user0"

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	cfg := raTest.MakeConfig(t)
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		require.NoError(t, err)
		_ = eventstore.Close(ctx)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices)

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
			deleteDevicesCtx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.owner,
			}))
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

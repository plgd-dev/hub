package service_test

import (
	"context"
	"sort"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRequestHandler_DeleteDevices(t *testing.T) {
	deviceID := dev0
	const user0 = "user0"

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	cfg := raTest.MakeConfig(t)
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Clear(ctx)
		require.NoError(t, errC)
		_ = eventstore.Close(ctx)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(cfg, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

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
					Owner:  user0,
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
					Owner:  user0,
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
					Owner:  user0,
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
				sort.Strings(tt.want.GetDeviceIds())
				sort.Strings(response.GetDeviceIds())
				require.Equal(t, tt.want.GetDeviceIds(), response.GetDeviceIds())
				require.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
			}
		})
	}
}

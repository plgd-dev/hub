package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	c2cConnectorTest "github.com/plgd-dev/hub/v2/cloud2cloud-connector/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testRequestHandlerGetDevices(t *testing.T, events store.Events) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.GetDevicesRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    []*pb.Device
	}{
		{
			name: "valid",
			args: args{
				req: &pb.GetDevicesRequest{},
			},
			want: []*pb.Device{
				{
					Types:      []string{types.DEVICE_CLOUD, device.ResourceType},
					Interfaces: []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
					Id:         deviceID,
					Name:       test.TestDeviceName,
					Metadata: &pb.Device_Metadata{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_ONLINE,
						},
					},
				},
			},
		},
	}

	const timeoutWithPull = testCfg.TEST_TIMEOUT + time.Second*10 // longer timeout is needed because of the 10s sleep in setUp
	ctx, cancel := context.WithTimeout(context.Background(), timeoutWithPull)
	defer cancel()
	tearDown := c2cConnectorTest.SetUpClouds(ctx, t, deviceID, events)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(c2cConnectorTest.GRPC_GATEWAY_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetDevices(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			devices := make([]*pb.Device, 0, 1)
			for {
				dev, err := client.Recv()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				assert.NotEmpty(t, dev.ProtocolIndependentId)
				dev.ProtocolIndependentId = ""
				if dev.GetMetadata().GetStatus() != nil {
					dev.GetMetadata().GetStatus().ValidUntil = 0
					dev.GetMetadata().GetStatus().ConnectionId = ""
				}
				devices = append(devices, dev)
			}
			test.CheckProtobufs(t, tt.want, devices, test.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestRequestHandlerGetDevices(t *testing.T) {
	type args struct {
		events store.Events
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "full pulling",
		},
		{
			name: "full events",
			args: args{
				events: store.Events{
					Devices:  events.AllDevicesEvents,
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "resource events + device,devices pulling",
			args: args{
				events: store.Events{
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "resource, device events + devices pulling",
			args: args{
				events: store.Events{
					Device:   events.AllDeviceEvents,
					Resource: events.AllResourceEvents,
				},
			},
		},
		{
			name: "device, devices events + resource pulling",
			args: args{
				events: store.Events{
					Devices: events.AllDevicesEvents,
					Device:  events.AllDeviceEvents,
				},
			},
		},
		{
			name: "pull resource, devices + static device events",
			args: args{
				events: store.Events{
					StaticDeviceEvents: true,
				},
			},
		},
		{
			name: "resource, devices events + static device events",
			args: args{
				events: store.Events{
					Devices:            events.AllDevicesEvents,
					Resource:           events.AllResourceEvents,
					StaticDeviceEvents: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRequestHandlerGetDevices(t, tt.args.events)
		})
	}
}

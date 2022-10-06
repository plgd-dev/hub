package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerDeleteDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.DeleteDevicesRequest
	}
	tests := []struct {
		name string
		args args
		want *pb.DeleteDevicesResponse
	}{
		{
			name: "not owned device",
			args: args{
				req: &pb.DeleteDevicesRequest{
					DeviceIdFilter: []string{"badId"},
				},
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: nil,
			},
		},
		{
			name: "all owned devices",
			args: args{
				req: &pb.DeleteDevicesRequest{
					DeviceIdFilter: []string{},
				},
			},
			want: &pb.DeleteDevicesResponse{
				DeviceIds: []string{deviceID},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.DeleteDevices(ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.DeviceIds, resp.DeviceIds)
		})
	}
}

func waitForOperationProcessedEvent(t *testing.T, subClient pb.GrpcGateway_SubscribeToEventsClient, corID string) {
	ev, err := subClient.Recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  corID,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
}

func waitForStopEvent(t *testing.T, subClient pb.GrpcGateway_SubscribeToEventsClient, deviceID, corID string) {
	ev, err := subClient.Recv()
	require.NoError(t, err)

	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  corID,
		Type: &pb.Event_DeviceUnregistered_{
			DeviceUnregistered: &pb.Event_DeviceUnregistered{
				DeviceIds: []string{deviceID},
			},
		},
	}
	pbTest.CmpEvent(t, expectedEvent, ev, "")
}

func TestRequestHandlerReconnectAfterDeleteDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	_, _ = test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())

	const correlationID = "device"
	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	err = subClient.Send(&pb.SubscribeToEvents{
		CorrelationId: correlationID,
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_UNREGISTERED,
				},
			},
		},
	})
	require.NoError(t, err)
	waitForOperationProcessedEvent(t, subClient, correlationID)

	resp, err := c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIdFilter: []string{deviceID},
	})
	require.NoError(t, err)
	require.Equal(t, []string{deviceID}, resp.DeviceIds)
	waitForStopEvent(t, subClient, deviceID, correlationID)
	err = subClient.CloseSend()
	require.NoError(t, err)

	// wait for device to detect lost connection, try to reconnect and handle unauthorized code
	time.Sleep(10 * time.Second)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.ACTIVE_COAP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
}

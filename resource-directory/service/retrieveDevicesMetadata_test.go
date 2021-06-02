package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
)

func cmpDeviceMetadataSnapshotTaken(t *testing.T, want []*events.DeviceMetadataSnapshotTaken, got []*events.DeviceMetadataSnapshotTaken) {
	require.Len(t, got, len(want))
	for idx := range want {
		got[idx].EventMetadata = nil
		if got[idx].GetStatus() != nil {
			got[idx].GetStatus().AuditContext = nil
			got[idx].GetStatus().EventMetadata = nil
			got[idx].GetStatus().GetStatus().ValidUntil = 0
		}
		test.CheckProtobufs(t, want[idx], got[idx], test.RequireToCheckFunc(require.Equal))

	}
}

func TestRequestHandler_RetrieveDevicesMetadata(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req pb.RetrieveDevicesMetadataRequest
	}
	tests := []struct {
		name    string
		args    args
		want    []*events.DeviceMetadataSnapshotTaken
		wantErr bool
	}{
		{
			name: "all",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{},
			},
			want: []*events.DeviceMetadataSnapshotTaken{
				{
					DeviceId: deviceID,
					Status: &events.DeviceMetadataUpdated{
						DeviceId: deviceID,
						Updated: &events.DeviceMetadataUpdated_Status{
							Status: &commands.ConnectionStatus{
								Value: commands.ConnectionStatus_ONLINE,
							},
						},
					},
				},
			},
		},
		{
			name: "filter one device",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					DeviceIdsFilter: []string{deviceID},
				},
			},
			want: []*events.DeviceMetadataSnapshotTaken{
				{
					DeviceId: deviceID,
					Status: &events.DeviceMetadataUpdated{
						DeviceId: deviceID,
						Updated: &events.DeviceMetadataUpdated_Status{
							Status: &commands.ConnectionStatus{
								Value: commands.ConnectionStatus_ONLINE,
							},
						},
					},
				},
			},
		},
		{
			name: "invalid deviceID",
			args: args{
				req: pb.RetrieveDevicesMetadataRequest{
					DeviceIdsFilter: []string{"abc"},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.RetrieveDevicesMetadata(ctx, &tt.args.req)
			require.NoError(t, err)
			values := make([]*events.DeviceMetadataSnapshotTaken, 0, 1)
			for {
				value, err := client.Recv()
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				} else {
					require.NoError(t, err)
					values = append(values, value)
				}
			}
			cmpDeviceMetadataSnapshotTaken(t, tt.want, values)
		})
	}
}

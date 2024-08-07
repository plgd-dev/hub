package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getAllEvents(ctx context.Context, t *testing.T, client pb.GrpcGatewayClient) []interface{} {
	events := make([]interface{}, 0, len(test.GetAllBackendResourceLinks()))
	c, err := client.GetEvents(ctx, &pb.GetEventsRequest{
		TimestampFilter: 0,
	})
	require.NoError(t, err)
	for {
		value, err := c.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		event := pbTest.GetWrappedEvent(value)
		require.NotNil(t, event)
		events = append(events, event)
	}
	return events
}

func TestRequestHandlerGetEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	beforeOnBoard := time.Now().UnixNano()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	events := getAllEvents(ctx, t, c)
	require.NotEmpty(t, events)

	type args struct {
		req *pb.GetEventsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantLen int
		wantErr bool
	}{
		{
			name: "None (timestamp filter)",
			args: args{
				&pb.GetEventsRequest{
					TimestampFilter: time.Now().UnixNano(),
				},
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "All (timestamp filter)",
			args: args{
				&pb.GetEventsRequest{
					TimestampFilter: beforeOnBoard,
				},
			},
			wantLen: len(events),
			wantErr: false,
		},
		{
			name: "All (device filter)",
			args: args{
				&pb.GetEventsRequest{
					DeviceIdFilter:  []string{deviceID},
					TimestampFilter: beforeOnBoard,
				},
			},
			wantLen: len(events),
			wantErr: false,
		},
		{
			name: "First resource (resource filter)",
			args: args{
				&pb.GetEventsRequest{
					ResourceIdFilter: []*pb.ResourceIdFilter{
						{
							ResourceId: commands.NewResourceID(deviceID, test.GetAllBackendResourceLinks()[0].Href),
						},
					},
					TimestampFilter: beforeOnBoard,
				},
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := c.GetEvents(ctx, tt.args.req)
			require.NoError(t, err)
			values := make([]*pb.GetEventsResponse, 0, 1)
			for {
				value, err := client.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, value)
			}

			require.Len(t, values, tt.wantLen)
			pbTest.CheckGetEventsResponse(t, values)
		})
	}
}

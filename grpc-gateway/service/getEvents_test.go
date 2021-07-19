package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getWrappedEvent(value *pb.GetEventsResponse) interface{} {
	if event := value.GetDeviceMetadataSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetDeviceMetadataUpdatePending(); event != nil {
		return event
	}
	if event := value.GetDeviceMetadataUpdated(); event != nil {
		return event
	}
	if event := value.GetResourceChanged(); event != nil {
		return event
	}
	if event := value.GetResourceCreatePending(); event != nil {
		return event
	}
	if event := value.GetResourceCreated(); event != nil {
		return event
	}
	if event := value.GetResourceDeletePending(); event != nil {
		return event
	}
	if event := value.GetResourceDeleted(); event != nil {
		return event
	}
	if event := value.GetResourceLinksPublished(); event != nil {
		return event
	}
	if event := value.GetResourceLinksSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetResourceLinksUnpublished(); event != nil {
		return event
	}
	if event := value.GetResourceRetrievePending(); event != nil {
		return event
	}
	if event := value.GetResourceRetrieved(); event != nil {
		return event
	}
	if event := value.GetResourceStateSnapshotTaken(); event != nil {
		return event
	}
	if event := value.GetResourceUpdatePending(); event != nil {
		return event
	}
	if event := value.GetResourceUpdated(); event != nil {
		return event
	}
	return nil
}

func getAllEvents(t *testing.T, client pb.GrpcGatewayClient, ctx context.Context) []interface{} {
	events := make([]interface{}, 0, len(test.GetAllBackendResourceLinks()))
	c, err := client.GetEvents(ctx, &pb.GetEventsRequest{
		TimemstampFilter: 0,
	})
	require.NoError(t, err)
	for {
		value, err := c.Recv()
		if err == io.EOF {
			break
		}
		event := getWrappedEvent(value)
		require.NotNil(t, event)
		events = append(events, event)
	}
	return events
}

func checkGetEventsResponse(t *testing.T, deviceId string, got []*pb.GetEventsResponse) {
	for _, value := range got {
		event := getWrappedEvent(value)
		r := reflect.ValueOf(event)
		const CheckMethodName = "CheckInitialized"
		m := r.MethodByName(CheckMethodName)
		if !m.IsValid() {
			require.Failf(t, "Invalid type", "Struct %T doesn't have %v method", event, CheckMethodName)
		}
		v := m.Call([]reflect.Value{})
		require.Len(t, v, 1)
		initialized := v[0].Bool()
		require.True(t, initialized)
	}
}

func TestRequestHandler_GetEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

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

	beforeOnBoard := time.Now().UnixNano()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	events := getAllEvents(t, c, ctx)
	require.True(t, len(events) > 0)

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
					TimemstampFilter: time.Now().UnixNano(),
				},
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "All (timestamp filter)",
			args: args{
				&pb.GetEventsRequest{
					TimemstampFilter: beforeOnBoard,
				},
			},
			wantLen: len(events),
			wantErr: false,
		},
		{
			name: "All (device filter)",
			args: args{
				&pb.GetEventsRequest{
					DeviceIdFilter:   []string{deviceID},
					TimemstampFilter: beforeOnBoard,
				},
			},
			wantLen: len(events),
			wantErr: false,
		},
		{
			name: "First resource (resource filter)",
			args: args{
				&pb.GetEventsRequest{
					ResourceIdFilter: []string{commands.NewResourceID(deviceID, test.GetAllBackendResourceLinks()[0].Href).ToString()},
					TimemstampFilter: beforeOnBoard,
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

			require.Len(t, values, tt.wantLen)
			checkGetEventsResponse(t, deviceID, values)
		})
	}
}

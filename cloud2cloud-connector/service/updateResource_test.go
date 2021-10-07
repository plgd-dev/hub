package service_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	c2cConnectorTest "github.com/plgd-dev/hub/cloud2cloud-connector/test"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	raEvents "github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testRequestHandlerUpdateResource(t *testing.T, events store.Events) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req *pb.UpdateResourceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *raEvents.ResourceUpdated
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: &raEvents.ResourceUpdated{
				ResourceId: commands.NewResourceID(deviceID, "/light/1"),
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "valid with interface",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId:        commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: &raEvents.ResourceUpdated{
				ResourceId: commands.NewResourceID(deviceID, "/light/1"),
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "revert update",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: "oic.if.baseline",
					ResourceId:        commands.NewResourceID(deviceID, "/light/1"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: &raEvents.ResourceUpdated{
				ResourceId: commands.NewResourceID(deviceID, "/light/1"),
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
				Status: commands.Status_OK,
			},
		},
		{
			name: "update RO-resource",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"di": "abc",
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid Href",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := c2cConnectorTest.SetUpClouds(ctx, t, deviceID, events)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

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
			got, err := c.UpdateResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got.GetData())
				got.GetData().AuditContext = nil
				got.GetData().EventMetadata = nil
				test.CheckProtobufs(t, tt.want, got.GetData(), test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

func TestRequestHandlerUpdateResource(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRequestHandlerUpdateResource(t, tt.args.events)
		})
	}
}

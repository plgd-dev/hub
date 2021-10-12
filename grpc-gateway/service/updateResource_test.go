package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/ocf"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func resourceUpdated(deviceID, href string) *events.ResourceUpdated {
	return &events.ResourceUpdated{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			CoapContentFormat: -1,
		},
		Status: commands.Status_OK,
	}
}

func TestRequestHandler_UpdateResourcesValues(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchID := "1"
	type args struct {
		req *pb.UpdateResourceRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *events.ResourceUpdated
		wantErr bool
	}{
		{
			name: "invalid Href",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeToLive",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightHref),
					TimeToLive: int64(99 * time.Millisecond),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update RO-resource",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.OCFResourceDeviceHref),
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
			name: "invalid update collection /switches",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"href": test.TestResourceSwitchesInstanceHref(switchID),
						}),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceLightHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
			},
			want: resourceUpdated(deviceID, test.TestResourceLightHref),
		},
		{
			name: "valid with interface",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: ocf.OC_IF_BASELINE,
					ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 2,
						}),
					},
				},
			},
			want: resourceUpdated(deviceID, test.TestResourceLightHref),
		},
		{
			name: "revert update",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceInterface: ocf.OC_IF_BASELINE,
					ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightHref),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 0,
						}),
					},
				},
			},
			want: resourceUpdated(deviceID, test.TestResourceLightHref),
		},
		{
			name: "update /switches/1",
			args: args{
				req: &pb.UpdateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, test.TestResourceSwitchesInstanceHref(switchID)),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"value": true,
						}),
					},
				},
			},
			want: &events.ResourceUpdated{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     test.TestResourceSwitchesInstanceHref(switchID),
				},
				Content: &commands.Content{
					CoapContentFormat: int32(message.AppOcfCbor),
					ContentType:       message.AppOcfCbor.String(),
					Data: test.EncodeToCbor(t, map[string]interface{}{
						"value": true,
					}),
				},
				Status: commands.Status_OK,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultServiceToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)
	time.Sleep(200 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.UpdateResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			test.CmpResourceUpdated(t, tt.want, got.GetData())
		})
	}
}

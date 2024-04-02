package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	extCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	type args struct {
		token    string
		deviceID string
		href     string
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/light/1 - method not allowd",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     "/ligh/1",
			},
			wantErr:     true,
			wantErrCode: codes.Code(extCodes.MethodNotAllowed),
		},
		{
			name: "/light/1 - permission denied",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     device.ResourceURI,
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "delete /switches/1",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     test.TestResourceSwitchesInstanceHref("1"),
			},
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := NewTestClient(t)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
	_, shutdownDevsim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevsim()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c.GrpcGatewayClient(), "1", "2", "3")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			var got interface{}
			err := c.DeleteResource(runctx, tt.args.deviceID, tt.args.href, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientBatchDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	switchIDs := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	type args struct {
		token    string
		deviceID string
		href     string
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/oic/res - Delete not supported",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     resources.ResourceURI,
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/switches/1 - Batch delete not supported",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     test.TestResourceSwitchesInstanceHref("1"),
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "/switches",
			args: args{
				token:    oauthTest.GetDefaultAccessToken(t),
				deviceID: deviceID,
				href:     test.TestResourceSwitchesHref,
			},
			want: func() interface{} {
				links := test.CollectionLinkRepresentations{}
				for _, switchID := range switchIDs {
					links = append(links, test.CollectionLinkRepresentation{
						Href:           test.TestResourceSwitchesInstanceHref(switchID),
						Representation: map[interface{}]interface{}{},
					})
				}
				return links
			}(),
		},
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	c := NewTestClient(t)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
	_, shutdown := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdown()
	test.AddDeviceSwitchResources(ctx, t, deviceID, c.GrpcGatewayClient(), switchIDs...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			var got test.CollectionLinkRepresentations
			err := c.DeleteResource(runctx, tt.args.deviceID, tt.args.href, &got, client.WithInterface(interfaces.OC_IF_B))
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.wantErrCode, status.Convert(err).Code())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

package client_test

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token    string
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with skip shadow",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow()},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "valid with interface and skip shadow",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []client.GetOption{client.WithSkipShadow(), client.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"n":  test.TestDeviceName,
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"rt": []interface{}{"oic.wk.con"},
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    oauthTest.GetServiceToken(t),
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			var got interface{}
			err := c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClient_GetResourceUnavaliable(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetServiceToken(t))

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, testCfg.GW_HOST, nil)
	defer shutdownDevSim()

	waitForDevice := func() {
		h := makeTestDevicesObservationHandler()
		id, err := c.ObserveDevices(ctx, h)
		require.NoError(t, err)
		defer func() {
			c.StopObservingDevices(ctx, id)
		}()

		for {
			var res client.DevicesObservationEvent
			select {
			case res = <-h.res:
			case <-ctx.Done():
				require.NoError(t, ctx.Err())
			}
			if res.Event == client.DevicesObservationEvent_ONLINE {
				var found bool
				for _, d := range res.DeviceIDs {
					if d == deviceID {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	waitForDevice()

	waitForResource := func() {
		h := makeTestDeviceResourcesObservationHandler()
		id, err := c.ObserveDeviceResources(ctx, deviceID, h)
		require.NoError(t, err)
		defer func() {
			err := c.StopObservingDevices(ctx, id)
			require.NoError(t, err)
		}()
		for {
			var res client.DeviceResourcesObservationEvent
			select {
			case res = <-h.res:
			case <-ctx.Done():
				require.NoError(t, ctx.Err())
			}
			if res.Event == client.DeviceResourcesObservationEvent_ADDED {
				var found bool
				for _, d := range res.Links {
					if d.DeviceId == deviceID && d.Href == "/oc/con" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	waitForResource()

	var v interface{}
	err := c.GetResource(ctx, deviceID, "/oc/con", &v)
	s, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unavailable, s.Code())
}

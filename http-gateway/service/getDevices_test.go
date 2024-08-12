package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func makeDefaultDevice(deviceID string) *pb.Device {
	return &pb.Device{
		Types:       []string{types.DEVICE_CLOUD, device.ResourceType},
		Interfaces:  []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		Id:          deviceID,
		Name:        test.TestDeviceName,
		ModelNumber: test.TestDeviceModelNumber,
		Metadata: &pb.Device_Metadata{
			Connection: &commands.Connection{
				Status:   commands.Connection_ONLINE,
				Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
			},
			TwinEnabled: true,
			TwinSynchronization: &commands.TwinSynchronization{
				State: commands.TwinSynchronization_IN_SYNC,
			},
		},
		OwnershipStatus: pb.Device_OWNED,
	}
}

func TestRequestHandlerGetDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		accept         string
		typeFilter     []string
		statusFilter   []pb.GetDevicesRequest_Status
		deviceIdFilter []string
	}
	tests := []struct {
		name string
		args args
		want []*pb.Device
	}{
		{
			name: "all devices",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
			},
			want: []*pb.Device{makeDefaultDevice(deviceID)},
		},
		{
			name: "offline devices",
			args: args{
				accept:       pkgHttp.ApplicationProtoJsonContentType,
				statusFilter: []pb.GetDevicesRequest_Status{pb.GetDevicesRequest_OFFLINE},
			},
		},
		{
			name: "invalid device id",
			args: args{
				accept:         pkgHttp.ApplicationProtoJsonContentType,
				deviceIdFilter: []string{"invalid"},
			},
		},
		{
			name: "single device",
			args: args{
				accept:         pkgHttp.ApplicationProtoJsonContentType,
				deviceIdFilter: []string{deviceID},
			},
			want: []*pb.Device{makeDefaultDevice(deviceID)},
		},
		{
			name: "invalid device type",
			args: args{
				accept:     pkgHttp.ApplicationProtoJsonContentType,
				typeFilter: []string{"invalid"},
			},
		},
		{
			name: "cloud device type",
			args: args{
				accept:     pkgHttp.ApplicationProtoJsonContentType,
				typeFilter: []string{types.DEVICE_CLOUD},
			},
			want: []*pb.Device{makeDefaultDevice(deviceID)},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	defer func() {
		_ = conn.Close()
	}()
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	toStringSlice := func(s []pb.GetDevicesRequest_Status) []string {
		sf := make([]string, 0, len(s))
		for _, v := range s {
			sf = append(sf, strconv.FormatInt(int64(v), 10))
		}
		return sf
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, uri.Devices, nil).Accept(tt.args.accept).AuthToken(token)
			rb.AddTypeFilter(tt.args.typeFilter).AddStatusFilter(toStringSlice(tt.args.statusFilter)).AddDeviceIdFilter(tt.args.deviceIdFilter)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			var devices []*pb.Device
			for {
				var dev pb.Device
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &dev)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				assert.NotEmpty(t, dev.GetProtocolIndependentId())
				devices = append(devices, &dev)
			}
			pbTest.CmpDeviceValues(t, tt.want, devices)
		})
	}
}

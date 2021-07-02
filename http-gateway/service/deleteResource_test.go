package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	testHttp "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_DeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		deviceID     string
		resourceHref string
		accept       string
	}
	tests := []struct {
		name        string
		args        args
		want        *events.ResourceDeleted
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "/light/2 - MethodNotAllowed",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/light/2",
				accept:       uri.ApplicationJsonPBContentType,
			},
			want: &events.ResourceDeleted{
				ResourceId: commands.NewResourceID(deviceID, "/light/2"),
				Status:     commands.Status_METHOD_NOT_ALLOWED,
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
			},
		},
		{
			name: "invalid Href",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/unknown",
				accept:       uri.ApplicationJsonPBContentType,
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/oic/d - PermissionDenied",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/oic/d",
				accept:       uri.ApplicationJsonPBContentType,
			},
			want: &events.ResourceDeleted{
				ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				Status:     commands.Status_FORBIDDEN,
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := testHttp.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodDelete, uri.DeviceResourceLink, nil).DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).AuthToken(token).Accept(tt.args.accept).Build()
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			marshaler := runtime.JSONPb{}
			decoder := marshaler.NewDecoder(resp.Body)

			var got events.ResourceDeleted
			err = Unmarshal(resp.StatusCode, decoder, &got)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrCode.String(), status.Convert(err).Code().String())
			} else {
				require.NoError(t, err)
				got.EventMetadata = nil
				got.AuditContext = nil
				test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_CreateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		req         *pb.CreateResourceRequest
		accept      string
		contentType string
	}
	tests := []struct {
		name        string
		args        args
		want        *events.ResourceCreated
		wantErr     bool
		wantErrCode codes.Code
	}{

		{
			name: "invalid Href",
			args: args{
				req: &pb.CreateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/unknown"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
				accept:      uri.ApplicationProtoJsonContentType,
				contentType: uri.ApplicationProtoJsonContentType,
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/oic/d - PermissionDenied - " + uri.ApplicationProtoJsonContentType,
			args: args{
				req: &pb.CreateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
				accept:      uri.ApplicationProtoJsonContentType,
				contentType: uri.ApplicationProtoJsonContentType,
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name: "/oic/d - PermissionDenied - " + message.AppJSON.String(),
			args: args{
				req: &pb.CreateResourceRequest{
					ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
					Content: &pb.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: test.EncodeToCbor(t, map[string]interface{}{
							"power": 1,
						}),
					},
				},
				accept:      uri.ApplicationProtoJsonContentType,
				contentType: message.AppJSON.String(),
			},
			wantErr:     true,
			wantErrCode: codes.PermissionDenied,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := getContentData(tt.args.req.GetContent(), tt.args.contentType)
			require.NoError(t, err)
			request := httpgwTest.NewRequest(http.MethodPost, uri.DeviceResourceLink, bytes.NewReader([]byte(data))).DeviceId(tt.args.req.GetResourceId().GetDeviceId()).ResourceHref(tt.args.req.GetResourceId().GetHref()).AuthToken(token).Accept(tt.args.accept).ContentType(tt.args.contentType).Build()
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			var got events.ResourceCreated
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
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

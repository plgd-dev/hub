package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
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
		req    *pb.CreateResourceRequest
		accept string
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
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
		{
			name: "/oic/d - PermissionDenied - without accept",
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
			},
			want: &events.ResourceCreated{
				ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				Status:     commands.Status_FORBIDDEN,
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
			},
			wantErrCode: codes.OK,
		},
		{
			name: "/oic/d - PermissionDenied - accept ocf-cbor",
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
				accept: message.AppOcfCbor.String(),
			},
			want: &events.ResourceCreated{
				ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				Status:     commands.Status_FORBIDDEN,
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
			},
			wantErrCode: codes.OK,
		},
		{
			name: "/oic/d - PermissionDenied - accept json",
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
				accept: message.AppJSON.String(),
			},
			want: &events.ResourceCreated{
				ResourceId: commands.NewResourceID(deviceID, "/oic/d"),
				Status:     commands.Status_FORBIDDEN,
				Content: &commands.Content{
					CoapContentFormat: -1,
				},
			},
			wantErrCode: codes.OK,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := New(t, MakeConfig(t))
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
			var m jsonpb.Marshaler
			data, err := m.MarshalToString(tt.args.req)
			require.NoError(t, err)
			request := httpgwTest.NewRequest(http.MethodPost, uri.DeviceResourceLink, bytes.NewReader([]byte(data))).DeviceId(tt.args.req.GetResourceId().GetDeviceId()).ResourceHref(tt.args.req.GetResourceId().GetHref()).AuthToken(token).AcceptContent(tt.args.accept).Build()
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

			var got events.ResourceCreated
			err = service.Unmarshal(resp.StatusCode, decoder, &got)
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

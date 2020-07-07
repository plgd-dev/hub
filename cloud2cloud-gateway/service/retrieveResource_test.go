package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/go-ocf/go-coap/v2/message"

	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/uri"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const TEST_TIMEOUT = time.Second * 20

func TestRequestHandler_RetrieveResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		uri    string
		accept string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + cloud.StatusHref,
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + cloud.StatusHref,
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				"rt":     []interface{}{"x.cloud.device.status"},
				"if":     []interface{}{"oic.if.baseline"},
				"online": true,
			},
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID + cloud.StatusHref,
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + cloud.StatusHref,
				accept: message.AppOcfCbor.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want: map[interface{}]interface{}{
				"rt":     []interface{}{"x.cloud.device.status"},
				"if":     []interface{}{"oic.if.baseline"},
				"online": true,
			},
		},
		{
			name: "JSON: " + uri.Devices + "/" + deviceID + "/light/1",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1",
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			},
		},
		{
			name: "CBOR: " + uri.Devices + "/" + deviceID + "/light/1",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1",
				accept: message.AppOcfCbor.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppOcfCbor.String(),
			want: map[interface{}]interface{}{
				"name":  "Light",
				"power": uint64(0),
				"state": false,
			},
		},
		{
			name: "notFound",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/notFound",
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusNotFound,
			wantContentType: "text/plain",
			want:            "cannot retrieve resource: cannot retrieve resource(deviceID: " + deviceID + ", Href: /notFound): cannot retrieve resource({" + deviceID + " /notFound}): cannot retrieve resources values: rpc error: code = NotFound desc = not found",
		},
		{
			name: "invalidAccept",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1",
				accept: "application/invalid",
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: "text/plain",
			want:            "cannot retrieve resource: cannot retrieve: invalid accept header([application/invalid])",
		},
		{
			name: "JSON: " + uri.Devices + "//" + deviceID + cloud.StatusHref + "/",
			args: args{
				uri:    "https://" + testCfg.C2C_GW_HOST + uri.Devices + "//" + deviceID + cloud.StatusHref + "/",
				accept: message.AppJSON.String(),
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				"rt":     []interface{}{"x.cloud.device.status"},
				"if":     []interface{}{"oic.if.baseline"},
				"online": true,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := test.NewHTTPRequest(http.MethodGet, tt.args.uri, nil).AddHeader("Accept", tt.args.accept).Build(ctx, t)
			resp := test.DoHTTPRequest(t, req)
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer resp.Body.Close()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			if tt.want != nil {
				var got interface{}
				readFrom := func(w io.Reader, v interface{}) error {
					return fmt.Errorf("not supported")
				}
				switch tt.wantContentType {
				case message.AppJSON.String():
					readFrom = json.ReadFrom
				case message.AppCBOR.String(), message.AppOcfCbor.String():
					readFrom = cbor.ReadFrom
				case "text/plain":
					readFrom = func(w io.Reader, v interface{}) error {
						b, err := ioutil.ReadAll(w)
						if err != nil {
							return err
						}
						val := reflect.ValueOf(v)
						if val.Kind() != reflect.Ptr {
							return fmt.Errorf("some: check must be a pointer")
						}
						val.Elem().Set(reflect.ValueOf(string(b)))
						return nil
					}
				}
				err = readFrom(resp.Body, &got)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

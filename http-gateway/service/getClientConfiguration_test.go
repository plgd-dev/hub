package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/service"
	rdTest "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
	testCfg "github.com/plgd-dev/cloud/test/config"
)

func TestRequestHandler_GetClientConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		want    *pb.ClientConfigurationResponse
	}{
		{
			name: "valid",
			want: rdTest.MakeConfig(t).ExposedCloudConfiguration.ToProto(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := New(t, MakeConfig(t))
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%v/api/v1/clientConfiguration", config.HTTP_GW_HOST), nil)
			require.NoError(t, err)
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

			var got pb.ClientConfigurationResponse
			err = service.Unmarshal(resp.StatusCode, decoder, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got.CloudCertificateAuthorities)
			got.CloudCertificateAuthorities = ""
			test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))

		})
	}
}

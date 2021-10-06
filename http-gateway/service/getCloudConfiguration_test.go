package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/v2/http-gateway/test"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	rdTest "github.com/plgd-dev/cloud/v2/resource-directory/test"
	"github.com/plgd-dev/cloud/v2/test"
	"github.com/plgd-dev/cloud/v2/test/config"
)

func TestRequestHandler_GetCloudConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedCloudConfiguration.ToProto()
	expected.CurrentTime = 0
	tests := []struct {
		name    string
		wantErr bool
		want    *pb.CloudConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.CloudConfiguration, nil).Accept("" /*uri.ApplicationProtoJsonContentType*/).Build()
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

			var got pb.CloudConfigurationResponse
			err = Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got.CloudCertificateAuthorities)
			got.CloudCertificateAuthorities = ""
			require.NotEqual(t, int64(0), got.CurrentTime)
			got.CurrentTime = 0
			test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))
		})
	}
}

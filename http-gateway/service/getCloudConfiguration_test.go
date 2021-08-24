package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	rdTest "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/cloud/test/config"
)

func TestRequestHandler_GetCloudConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedCloudConfiguration.ToProto(config.MakeAuthURL())
	expected.CurrentTime = 0
	expected.DeviceOnboardingCodeUrl = ""
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
			require.NotEmpty(t, got.DeviceOnboardingCodeUrl)
			expected.DeviceOnboardingCodeUrl = ""
			test.CheckProtobufs(t, tt.want, &got, test.RequireToCheckFunc(require.Equal))
		})
	}
}

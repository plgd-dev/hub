package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/http-gateway/test"
	"github.com/plgd-dev/hub/http-gateway/uri"
	rdTest "github.com/plgd-dev/hub/resource-directory/test"
	"github.com/plgd-dev/hub/test/config"
	pbTest "github.com/plgd-dev/hub/test/pb"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetHubConfiguration(t *testing.T) {
	expected := rdTest.MakeConfig(t).ExposedHubConfiguration.ToProto()
	expected.CurrentTime = 0
	tests := []struct {
		name string
		want *pb.HubConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.HubConfiguration, nil).Build()
			resp := httpgwTest.HTTPDo(t, request)
			defer func() {
				_ = resp.Body.Close()
			}()
			var got pb.HubConfigurationResponse
			err := Unmarshal(resp.StatusCode, resp.Body, &got)
			require.NoError(t, err)
			pbTest.CmpHubConfigurationResponse(t, tt.want, &got)
		})
	}
}

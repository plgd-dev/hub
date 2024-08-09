package service_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/config"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestRequestHandlerGetHubConfiguration(t *testing.T) {
	rdCfg := rdTest.MakeConfig(t)
	rdCfg.ExposedHubConfiguration.Authority = "https://" + config.OAUTH_SERVER_HOST + "?escape=test&test=escape"
	httpCfg := httpgwTest.MakeConfig(t, true)
	expected := rdCfg.ExposedHubConfiguration.ToProto(config.HubID())
	expected.CurrentTime = 0
	expected.WebOauthClient = httpCfg.UI.WebConfiguration.WebOAuthClient.ToProto()
	expected.DeviceOauthClient = httpCfg.UI.WebConfiguration.DeviceOAuthClient.ToProto()
	expected.M2MOauthClient = httpCfg.UI.WebConfiguration.M2MOAuthClient.ToProto()
	expected.HttpGatewayAddress = httpCfg.UI.WebConfiguration.HTTPGatewayAddress
	expected.Ui = &pb.UIConfiguration{
		Visibility:                httpCfg.UI.WebConfiguration.Visibility.ToProto(),
		DeviceProvisioningService: httpCfg.UI.WebConfiguration.DeviceProvisioningService,
		SnippetService:            httpCfg.UI.WebConfiguration.SnippetService,
	}
	tests := []struct {
		name   string
		accept string
		want   *pb.HubConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
		{
			name:   "valid configuration",
			accept: pkgHttp.ApplicationProtoJsonContentType,
			want:   expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t, service.WithRDConfig(rdCfg))
	defer tearDown()

	shutdownHttp := httpgwTest.New(t, httpCfg)
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.HubConfiguration, nil).Accept(tt.accept).Build()
			resp := httpgwTest.HTTPDo(t, request)
			defer func() {
				_ = resp.Body.Close()
			}()
			var got pb.HubConfigurationResponse
			err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			require.NoError(t, err)
			pbTest.CmpHubConfigurationResponse(t, tt.want, &got)
		})
	}
}

func TestRequestHandlerGetHubConfigurationWithoutM2MOAuthClient(t *testing.T) {
	rdCfg := rdTest.MakeConfig(t)
	rdCfg.ExposedHubConfiguration.Authority = "https://" + config.OAUTH_SERVER_HOST + "?escape=test&test=escape"
	httpCfg := httpgwTest.MakeConfig(t, true)
	httpCfg.UI.WebConfiguration.M2MOAuthClient = nil
	expected := rdCfg.ExposedHubConfiguration.ToProto(config.HubID())
	expected.CurrentTime = 0
	expected.WebOauthClient = httpCfg.UI.WebConfiguration.WebOAuthClient.ToProto()
	expected.DeviceOauthClient = httpCfg.UI.WebConfiguration.DeviceOAuthClient.ToProto()
	expected.M2MOauthClient = nil
	expected.HttpGatewayAddress = httpCfg.UI.WebConfiguration.HTTPGatewayAddress
	expected.Ui = &pb.UIConfiguration{
		Visibility:                httpCfg.UI.WebConfiguration.Visibility.ToProto(),
		DeviceProvisioningService: httpCfg.UI.WebConfiguration.DeviceProvisioningService,
		SnippetService:            httpCfg.UI.WebConfiguration.SnippetService,
	}
	tests := []struct {
		name   string
		accept string
		want   *pb.HubConfigurationResponse
	}{
		{
			name: "valid",
			want: expected,
		},
		{
			name:   "valid configuration",
			accept: pkgHttp.ApplicationProtoJsonContentType,
			want:   expected,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t, service.WithRDConfig(rdCfg))
	defer tearDown()

	shutdownHttp := httpgwTest.New(t, httpCfg)
	defer shutdownHttp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, uri.HubConfiguration, nil).Accept(tt.accept).Build()
			resp := httpgwTest.HTTPDo(t, request)
			defer func() {
				_ = resp.Body.Close()
			}()
			var got pb.HubConfigurationResponse
			err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
			require.NoError(t, err)
			pbTest.CmpHubConfigurationResponse(t, tt.want, &got)
		})
	}
}

package service_test

import (
	"context"
	"net/http"
	"testing"

	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestHTTPMethodValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	httpgwCfg := httpgwTest.MakeConfig(t, false)
	shutdownHttp := httpgwTest.New(t, httpgwCfg)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)

	type args struct {
		method string
		uri    string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
	}{
		{
			name:         "invalid method",
			args:         args{method: "INVALID", uri: uri.Devices},
			wantHTTPCode: http.StatusUnauthorized,
		},
		{
			name:         "inaccessible URI",
			args:         args{method: http.MethodGet, uri: "/invalid/uri"},
			wantHTTPCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(tt.args.method, tt.args.uri, nil).Accept(uri.ApplicationProtoJsonContentType).AuthToken(token)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)
		})
	}
}

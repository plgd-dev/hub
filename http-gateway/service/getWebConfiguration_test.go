package service_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	httpgwService "github.com/plgd-dev/hub/v2/http-gateway/service"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpAPI(t *testing.T) {
	tests := []struct {
		val string
		ok  bool
	}{
		{"/api", false},
		{"/api/", false},
		{"/api/a", false},
		{"/a", true},
		{"/ap", true},
		{"/bpi", true},
		{"/bpi/", true},
		{"/bpi/a", true},
		{"/abi", true},
		{"/abi/", true},
		{"/abi/a", true},
		{"/apb", true},
		{"/apb/", true},
		{"/apb/a", true},
		{uri.WebConfiguration, true},
		{"/abcdefg/asdsa", true},
	}
	v := regexp.MustCompile(httpgwService.AuthorizationWhiteListedEndpointsRegexp)
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			assert.Equal(t, tt.ok, v.MatchString(tt.val))
		})
	}
}

func TestRequestHandlerGetWebConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	tests := []struct {
		name         string
		token        string
		enableUI     bool
		want         httpgwService.WebConfiguration
		wantErr      bool
		wantHTTPCode int
	}{
		{
			name:         "disabled UI",
			wantErr:      true,
			wantHTTPCode: http.StatusUnauthorized,
		},
		{
			name:         "valid configuration",
			enableUI:     true,
			want:         httpgwTest.MakeWebConfigurationConfig(),
			wantHTTPCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpgwCfg := httpgwTest.MakeConfig(t, tt.enableUI)
			shutdownHttp := httpgwTest.New(t, httpgwCfg)
			defer shutdownHttp()

			rb := httpgwTest.NewRequest(http.MethodGet, uri.WebConfiguration, nil)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got httpgwService.WebConfiguration
			err := httpTest.UnmarshalJson(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRequestHandlerGetWebDirectory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*2)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	httpgwCfg := httpgwTest.MakeConfig(t, true)
	shutdownHttp := httpgwTest.New(t, httpgwCfg)
	defer shutdownHttp()

	wwwRoot := os.Getenv("TEST_HTTP_GW_WWW_ROOT")

	tests := []struct {
		name     string
		uri      string
		wantFile string
	}{
		{
			name:     "invalid file - fallback to index.html",
			uri:      "/test.html",
			wantFile: wwwRoot + "/index.html",
		},
		{
			name:     "robots.txt",
			uri:      "/robots.txt",
			wantFile: wwwRoot + "/robots.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, tt.uri, nil)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()

			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			want, err := os.ReadFile(tt.wantFile)
			require.NoError(t, err)

			require.Equal(t, want, got)
		})
	}
}

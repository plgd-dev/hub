//go:build test
// +build test

package service_test

import (
	"context"
	"testing"
	"time"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

type TestCoapRefreshTokenResponse struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"refreshtoken"`
	ExpiresIn    int64  `json:"-"`
}

func TestRefreshTokenHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid deviceID`, nil}, true},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle refresh token for unknown: cbor: cannot unmarshal`, nil}, true},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": "refresh-token"}`, nil}, output{coapCodes.BadRequest, `invalid userId`, nil}, true},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "` + AuthorizationUserID + `"}`, nil}, output{coapCodes.BadRequest, `invalid refreshToken`, nil}, true},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserID + `", "refreshtoken":"refresh-token"}`, nil}, output{coapCodes.Changed, TestCoapRefreshTokenResponse{RefreshToken: AuthorizationRefreshToken}, nil}, false},
	}

	shutdown := setUp(t)
	defer shutdown()

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
			if co == nil {
				return
			}
			testSignUp(t, CertIdentity, co)
			defer func() {
				_ = co.Close()
			}()
			testPostHandler(t, uri.RefreshToken, test, co)
		}
		t.Run(test.name, tf)
	}
}

type TestCoapRefreshTokenResponseRetry struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	ExpiresIn    int64  `json:"-"`
}

func TestRefreshTokenHandlerWithRetry(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].Config.ClientID = oauthTest.ClientTestRestrictedAuth
	shutdown := setUp(t, testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()
	testSignUp(t, CertIdentity, co)

	testRefreshToken := testEl{
		name: "Retry",
		in:   input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserID + `", "refreshtoken":"refresh-token"}`, nil},
		out:  output{coapCodes.Changed, TestCoapRefreshTokenResponseRetry{}, nil},
	}

	const retryCount = 3
	for i := 0; i < retryCount; i++ {
		testPostHandler(t, uri.RefreshToken, testRefreshToken, co)
		time.Sleep(time.Second)
	}
}

func TestRefreshTokenWithOAuthNotWorking(t *testing.T) {
	test := testEl{"ServiceUnavailable", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserID + `", "refreshtoken":"refresh-token"}`, nil}, output{coapCodes.ServiceUnavailable, `ServiceUnavailable`, nil}, true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	cfg := oauthTest.MakeConfig(t)
	oauthShutdown := oauthTest.New(t, cfg)
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].HTTP.Timeout = time.Second
	shutdown := testService.SetUpServices(ctx, t, service.SetUpServicesMachine2MachineOAuth|testService.SetUpServicesId|testService.SetUpServicesCoapGateway|testService.SetUpServicesResourceAggregate|testService.SetUpServicesResourceDirectory, testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	testSignUp(t, CertIdentity, co)
	defer func() {
		_ = co.Close()
	}()
	oauthShutdown()
	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()
	s, err := listener.New(config.MakeListenerConfig(cfg.APIs.HTTP.Connection.Addr), fileWatcher, log.Get(), noop.NewTracerProvider())
	require.NoError(t, err)
	defer func() {
		err = s.Close()
		require.NoError(t, err)
	}()
	testPostHandler(t, uri.RefreshToken, test, co)
}

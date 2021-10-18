package service_test

import (
	"testing"
	"time"

	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	coapgwTest "github.com/plgd-dev/hub/coap-gateway/test"
	"github.com/plgd-dev/hub/coap-gateway/uri"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
)

type TestCoapRefreshTokenResponse struct {
	ExpiresIn    int64  `json:"-"`
	AccessToken  string `json:"-"`
	RefreshToken string `json:"refreshtoken"`
}

func TestRefreshTokenHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid deviceID`, nil}, true},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle refresh token for unknown: cbor: cannot unmarshal`, nil}, true},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": "123"}`, nil}, output{coapCodes.BadRequest, `invalid userId`, nil}, true},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "` + AuthorizationUserId + `"}`, nil}, output{coapCodes.BadRequest, `invalid refreshToken`, nil}, true},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "refreshtoken":"123"}`, nil}, output{coapCodes.Changed, TestCoapRefreshTokenResponse{RefreshToken: AuthorizationRefreshToken}, nil}, false},
	}

	shutdown := setUp(t)
	defer shutdown()

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, config.GW_HOST, "")
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
	ExpiresIn    int64  `json:"-"`
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
}

func TestRefreshTokenHandlerWithRetry(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].Config.ClientID = oauthTest.ClientTestRestrictedAuth
	shutdown := setUp(t, coapgwCfg)
	defer shutdown()

	co := testCoapDial(t, config.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()
	testSignUp(t, CertIdentity, co)

	testRefreshToken := testEl{
		name: "Retry",
		in:   input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "refreshtoken":"123"}`, nil},
		out:  output{coapCodes.Changed, TestCoapRefreshTokenResponseRetry{}, nil},
	}

	const retryCount = 3
	for i := 0; i < retryCount; i++ {
		testPostHandler(t, uri.RefreshToken, testRefreshToken, co)
		time.Sleep(time.Second)
	}
}

package service_test

import (
	"testing"
	"time"

	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/cloud/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
)

type TestCoapSignUpResponse struct {
	AccessToken  string `json:"-"`
	RedirectURI  string `json:"redirecturi"`
	ExpiresIn    uint64 `json:"-"`
	RefreshToken string `json:"refreshtoken"`
	UserID       string `json:"uid"`
}

func TestSignUpPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()
	codeEl := oauthTest.GetDefaultDeviceAuthorizationCode(t, "")

	tbl := []testEl{
		{"BadRequest (invalid)", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid device id`, nil}, true},
		{"BadRequest (invalid access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}, true},
		{"Changed", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + config.DEVICE_PROVIDER + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: "1"}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, config.GW_HOST, "")
			if co == nil {
				return
			}
			defer func() {
				_ = co.Close()
			}()
			testPostHandler(t, uri.SignUp, test, co)
		}
		t.Run(test.name, tf)
	}
}

type TestCoapSignUpResponseRetry struct {
	AccessToken  string `json:"-"`
	RedirectURI  string `json:"-"`
	ExpiresIn    uint64 `json:"-"`
	RefreshToken string `json:"-"`
	UserID       string `json:"uid"`
}

func TestSignUpPostHandlerWithRetry(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].Config.ClientID = service.ClientTestRestrictedAuth
	shutdown := setUp(t, coapgwCfg)
	defer shutdown()
	codeEl := oauthTest.GetDefaultDeviceAuthorizationCode(t, "")

	co := testCoapDial(t, config.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	testSignUp := testEl{
		name: "Retry",
		in:   input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + config.DEVICE_PROVIDER + `"}`, nil},
		out:  output{coapCodes.Changed, TestCoapSignUpResponseRetry{UserID: "1"}, nil},
	}

	const retryCount = 3
	for i := 0; i < retryCount; i++ {
		testPostHandler(t, uri.SignUp, testSignUp, co)
		time.Sleep(time.Second)
	}
}

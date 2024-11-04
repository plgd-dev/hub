//go:build test
// +build test

package service_test

import (
	"testing"
	"time"

	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	oauthUri "github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

type TestCoapSignUpResponse struct {
	AccessToken  string `json:"-"`
	RedirectURI  string `json:"redirecturi"`
	RefreshToken string `json:"refreshtoken"`
	UserID       string `json:"uid"`
	ExpiresIn    uint64 `json:"-"`
}

func TestSignUpPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()
	codeEl := oauthTest.GetDefaultDeviceAuthorizationCode(t, "")

	tbl := []testEl{
		{"BadRequest (invalid)", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid device id`, nil}, true},
		{"BadRequest (missing access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `"}`, nil}, output{coapCodes.BadRequest, `invalid authorization code`, nil}, true},
		{"BadRequest (invalid access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}, true},
		{"BadRequest (unknown provider)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "test"}`, nil}, output{coapCodes.Unauthorized, `unknown authorization provider`, nil}, true},
		{"Changed", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + config.DEVICE_PROVIDER + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: AuthorizationRefreshToken, UserID: "1"}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, "", true, false, time.Now().Add(time.Minute))
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
	RefreshToken string `json:"-"`
	UserID       string `json:"uid"`
	ExpiresIn    uint64 `json:"-"`
}

func TestSignUpPostHandlerWithRetry(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].Config.ClientID = oauthTest.ClientTestRestrictedAuth
	shutdown := setUp(t, testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()
	codeEl := oauthTest.GetDefaultDeviceAuthorizationCode(t, "")

	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
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

func TestSignUpClientCredentialPostHandler(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.Authorization.Providers[0].GrantType = oauth.ClientCredentials
	err := coapgwCfg.Validate()
	require.Error(t, err)
	coapgwCfg.APIs.COAP.Authorization.OwnerClaim = oauthUri.OwnerClaimKey
	err = coapgwCfg.Validate()
	require.NoError(t, err)
	shutdown := setUp(t, testService.WithCOAPGWConfig(coapgwCfg))
	defer shutdown()
	codeEl := oauthTest.GetDefaultAccessToken(t)

	tbl := []testEl{
		{"BadRequest (invalid)", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid device id`, nil}, true},
		{"BadRequest (invalid access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}, true},
		{"Changed", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + config.DEVICE_PROVIDER + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponseRetry{UserID: "1"}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
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

package service_test

import (
	"testing"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
)

type TestCoapSignInResponse struct {
	ExpiresIn uint64 `json:"-"`
}

func TestSignInPostHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{"login": true, "uid": "0", "accesstoken":"` + oauthTest.DeviceAccessToken + `" }`, nil}, output{coapCodes.BadRequest, `invalid DeviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123, "login": true}`, nil}, output{coapCodes.BadRequest, `cannot handle sign in: cbor: cannot unmarshal positive integer`, nil}},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid UserId`, nil}},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid AccessToken`, nil}},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
	testPostHandler(t, uri.SignUp, signUpEl, co)

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.SignIn, test, co)
			testPostHandler(t, uri.SecureSignIn, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestSignOutPostHandler(t *testing.T) {
	tbl := []testEl{
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "login": false }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"signUp", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
	t.Run(signUpEl.name, func(t *testing.T) {
		testPostHandler(t, uri.SignUp, signUpEl, co)
	})
	signInEl := testEl{"signIn", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}}
	t.Run(signInEl.name, func(t *testing.T) {
		testPostHandler(t, uri.SignIn, signInEl, co)
	})

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.SignIn, test, co)
			testPostHandler(t, uri.SecureSignIn, test, co)
		}
		t.Run(test.name, tf)
	}
}

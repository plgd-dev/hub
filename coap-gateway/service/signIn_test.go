package service_test

import (
	"testing"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	authTest "github.com/go-ocf/cloud/authorization/test"
	coapgwTest "github.com/go-ocf/cloud/coap-gateway/test"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	raTest "github.com/go-ocf/cloud/resource-aggregate/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
)

type TestCoapSignInResponse struct {
	ExpiresIn uint64 `json:"-"`
}

func TestSignInPostHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{"login": true }`, nil}, output{coapCodes.BadRequest, `invalid deviceID`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123, "login": true}`, nil}, output{coapCodes.BadRequest, `cannot handle sign in: cbor: cannot unmarshal positive integer`, nil}},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid userId`, nil}},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid accessToken`, nil}},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.UserToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}},
	}

	defer authTest.SetUp(t)
	defer raTest.SetUp(t)
	defer coapgwTest.SetUp(t)

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
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
		{"Changed0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.UserToken + `", "login": false }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}},
	}

	defer authTest.SetUp(t)
	defer raTest.SetUp(t)
	defer coapgwTest.SetUp(t)

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"signUp", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
	t.Run(signUpEl.name, func(t *testing.T) {
		testPostHandler(t, uri.SignUp, signUpEl, co)
	})
	signInEl := testEl{"signIn", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.UserToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}}
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

package service_test

import (
	"testing"

	oauthTest "github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
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
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid DeviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}},
		{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: "1"}, nil}},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.SignUp, test, co)
			testPostHandler(t, uri.SecureSignUp, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestSignOffHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.DELETE, `{}`, nil}, output{coapCodes.BadRequest, "invalid 'di'", nil}},
		{"BadRequest1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity}}, output{coapCodes.BadRequest, "invalid 'accesstoken'", nil}},
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + oauthTest.DeviceAccessToken, "uid=1"}}, output{coapCodes.Deleted, nil, nil}},
	}

	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
	for _, test := range tbl {
		tf := func(t *testing.T) {
			// create record for signUp
			testPostHandler(t, uri.SignUp, signUpEl, co)
			// delete record for signUp
			testPostHandler(t, uri.SignUp, test, co)

			// create record for secureSignUp
			testPostHandler(t, uri.SecureSignUp, signUpEl, co)
			// delete record for secureSignUp
			testPostHandler(t, uri.SecureSignUp, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestSignOffWithSignInHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"signUp", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
	testPostHandler(t, uri.SignUp, signUpEl, co)

	signInEl := testEl{"signIn", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.DeviceAccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}}
	testPostHandler(t, uri.SignIn, signInEl, co)

	tbl := []testEl{
		{"Deleted", input{coapCodes.DELETE, `{}`, nil}, output{coapCodes.Deleted, nil, nil}},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			// delete record for signUp
			testPostHandler(t, uri.SignUp, test, co)
		}
		t.Run(test.name, tf)
	}
}

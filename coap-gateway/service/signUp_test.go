package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	oauthTest "github.com/plgd-dev/cloud/oauth-server/test"
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
	shutdown := setUp(t)
	defer shutdown()
	codeEl := oauthTest.GetDeviceAuthorizationCode(t)

	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid DeviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}},
		{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + "auth0" + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: "1"}, nil}},
	}

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
	shutdown := setUp(t)
	defer shutdown()
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.DELETE, `{}`, nil}, output{coapCodes.BadRequest, "invalid 'di'", nil}},
		{"BadRequest1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity}}, output{coapCodes.BadRequest, "invalid 'accesstoken'", nil}},
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + "device", "uid=1"}}, output{coapCodes.Deleted, nil, nil}},
	}

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()
	codeEl := oauthTest.GetDeviceAuthorizationCode(t)
	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + "auth0" + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: AuthorizationUserId}, nil}}
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

package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
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
	codeEl := oauthTest.GetDeviceAuthorizationCode(t)

	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid DeviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}},
		{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + codeEl + `", "authprovider": "` + "plgd" + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserID: "1"}, nil}},
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

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer co.Close()

	signUpResp := testSignUp(t, CertIdentity, co)

	tbl := []testEl{
		{"BadRequest0", input{coapCodes.DELETE, `{}`, nil}, output{coapCodes.BadRequest, "invalid 'di'", nil}},
		{"BadRequest1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity}}, output{coapCodes.BadRequest, "invalid 'accesstoken'", nil}},
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + signUpResp.AccessToken, "uid=" + signUpResp.UserID}}, output{coapCodes.Deleted, nil, nil}},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			// delete record for signUp
			testPostHandler(t, uri.SignUp, test, co)
		}
		t.Run(test.name, tf)
	}

	signUpResp = testSignUp(t, CertIdentity, co)
	for _, test := range tbl {
		tf := func(t *testing.T) {
			// delete record for secureSignUp
			testPostHandler(t, uri.SecureSignUp, test, co)
		}
		t.Run(test.name, tf)
	}
}

package service_test

import (
	"testing"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	testCfg "github.com/go-ocf/cloud/test/config"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
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
		/* TODO: coap.URIQuery param has limit to 255 bytes, but jwt token has around 460. Token cannot be send by coap.URIQuery
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + oauthTest.UserToken}}, output{coapCodes.Deleted, nil, nil}},
		{"Deleted1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + oauthTest.UserToken, "uid=1"}}, output{coapCodes.Deleted, nil, nil}},
		*/
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

package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/stretchr/testify/require"
)

type TestCoapSignInResponse struct {
	ExpiresIn uint64 `json:"-"`
}

func TestSignInPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	signUpResp := testSignUp(t, CertIdentity, co)
	err := co.Close()
	require.NoError(t, err)

	tbl := []testEl{
		{"BadRequest (invalid request)", input{coapCodes.POST, `{"login": true}`, nil}, output{coapCodes.BadRequest, `invalid user id`, nil}, true},
		{"BadRequest (invalid userID)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.InternalServerError, `doesn't match userID`, nil}, true},
		{"BadRequest (missing access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid access token`, nil}, true},
		{"BadRequest (invalid access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123, "login": true}`, nil}, output{coapCodes.BadRequest, `cannot handle sign in: cbor: cannot unmarshal positive integer`, nil}, true},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, testCfg.GW_HOST)
			if co == nil {
				return
			}
			defer func() {
				_ = co.Close()
			}()
			testPostHandler(t, uri.SignIn, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestSignOutPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST)
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	signUpResp := testSignUp(t, CertIdentity, co)
	testSignIn(t, signUpResp, co)

	tbl := []testEl{
		{"Changed (uid from ctx)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": false }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}, false},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": false }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.SignIn, test, co)
		}
		t.Run(test.name, tf)
	}
}

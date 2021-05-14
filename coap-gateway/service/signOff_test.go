package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
)

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
		{"BadRequest0", input{coapCodes.DELETE, `{}`, nil}, output{coapCodes.BadRequest, "invalid di('')", nil}},
		{"BadRequest1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity}}, output{coapCodes.BadRequest, "invalid accesstoken('')", nil}},
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + signUpResp.AccessToken, "uid=" + signUpResp.UserID}}, output{coapCodes.Deleted, nil, nil}},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			// delete record for signUp
			testPostHandler(t, uri.SignUp, test, co)
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

	testSignUpIn(t, CertIdentity, co)

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

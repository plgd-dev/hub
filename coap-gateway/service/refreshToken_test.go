package service_test

import (
	"testing"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	testCfg "github.com/plgd-dev/cloud/test/config"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
)

type TestCoapRefreshTokenResponse struct {
	ExpiresIn    int64  `json:"-"`
	AccessToken  string `json:"-"`
	RefreshToken string `json:"refreshtoken"`
}

func Test_refreshTokenHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid deviceID`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle refresh token: cbor: cannot unmarshal`, nil}},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": "123"}`, nil}, output{coapCodes.BadRequest, `invalid userId`, nil}},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "` + AuthorizationUserId + `"}`, nil}, output{coapCodes.BadRequest, `invalid refreshToken`, nil}},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "refreshtoken":"123" }`, nil}, output{coapCodes.Changed, TestCoapRefreshTokenResponse{RefreshToken: AuthorizationRefreshToken}, nil}},
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
			testPostHandler(t, uri.RefreshToken, test, co)
		}
		t.Run(test.name, tf)
	}
}

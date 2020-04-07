package service

import (
	"testing"

	"github.com/go-ocf/ocf-cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

type TestCoapRefreshTokenResponse struct {
	ExpiresIn    int64  `json:"-"`
	AccessToken  string `json:"-"`
	RefreshToken string `json:"refreshtoken"`
}

func Test_refreshTokenHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid deviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle refresh token: cbor: cannot unmarshal`, nil}},
		{"BadRequest2", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "refreshtoken": "123"}`, nil}, output{coapCodes.BadRequest, `invalid userId`, nil}},
		{"BadRequest3", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "` + AuthorizationUserId + `"}`, nil}, output{coapCodes.BadRequest, `invalid refreshToken`, nil}},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "refreshtoken":"123" }`, nil}, output{coapCodes.Changed, TestCoapRefreshTokenResponse{RefreshToken: AuthorizationRefreshToken}, nil}},
	}

	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, resourceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	for _, test := range tbl {
		tf := func(t *testing.T) {
			testPostHandler(t, uri.RefreshToken, test, co)
			testPostHandler(t, uri.SecureRefreshToken, test, co)
		}
		t.Run(test.name, tf)
	}
}

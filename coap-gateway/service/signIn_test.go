package service

import (
	"testing"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
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

	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"
	deviceDB := t.Name() + "_deviceDB"
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, deviceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserId: AuthorizationUserId}, nil}}
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

	var config Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.AuthServerAddr = "localhost:12345"
	config.ResourceAggregateAddr = "localhost:12348"
	config.ResourceDirectoryAddr = "localhost:12349"
	deviceDB := t.Name() + "_deviceDB"
	resourceDB := t.Name() + "_resourceDB"

	shutdownSA := testCreateAuthServer(t, config.AuthServerAddr)
	defer shutdownSA()
	shutdownRA := testCreateResourceAggregate(t, deviceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownRA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"signUp", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserId: AuthorizationUserId}, nil}}
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

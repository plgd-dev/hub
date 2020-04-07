package service

import (
	"testing"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

type TestCoapSignUpResponse struct {
	AccessToken  string `json:"-"`
	RedirectUri  string `json:"redirecturi"`
	ExpiresIn    uint64 `json:"-"`
	RefreshToken string `json:"refreshtoken"`
	UserId       string `json:"uid"`
}

func TestSignUpPostHandler(t *testing.T) {
	tbl := []testEl{
		{"BadRequest0", input{coapCodes.POST, `{}`, nil}, output{coapCodes.BadRequest, `invalid deviceId`, nil}},
		{"BadRequest1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123}`, nil}, output{coapCodes.BadRequest, `cannot handle sign up: cbor: cannot unmarshal positive`, nil}},
		{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserId: "1"}, nil}},
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
	shutdownDA := testCreateResourceAggregate(t, deviceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownDA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
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
		/* TODO: coap.URIQuery param has limit to 255 bytes, but jwt token has arround 460. Token cannot be send by coap.URIQuery
		{"Deleted0", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + oauthTest.UserToken}}, output{coapCodes.Deleted, nil, nil}},
		{"Deleted1", input{coapCodes.DELETE, `{}`, []string{"di=" + CertIdentity, "accesstoken=" + oauthTest.UserToken, "uid=1"}}, output{coapCodes.Deleted, nil, nil}},
		*/
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
	shutdownDA := testCreateResourceAggregate(t, deviceDB, config.ResourceAggregateAddr, config.AuthServerAddr)
	defer shutdownDA()
	shutdownGW := testCreateCoapGateway(t, resourceDB, config)
	defer shutdownGW()

	co := testCoapDial(t, config.Addr, config.Net)
	if co == nil {
		return
	}
	defer co.Close()

	signUpEl := testEl{"Changed0", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserId: AuthorizationUserId}, nil}}
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

package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	test "github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type TestCoapSignInResponse struct {
	ExpiresIn uint64 `json:"-"`
}

func TestSignInPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	signUpResp := testSignUp(t, CertIdentity, co)
	err := co.Close()
	require.NoError(t, err)

	tbl := []testEl{
		{"BadRequest (invalid request)", input{coapCodes.POST, `{"login": true}`, nil}, output{coapCodes.BadRequest, `invalid device id`, nil}, true},
		{"BadRequest (invalid userID)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.InternalServerError, `doesn't match userID`, nil}, true},
		{"BadRequest (missing access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid": "0", "login": true }`, nil}, output{coapCodes.BadRequest, `invalid access token`, nil}, true},
		{"BadRequest (invalid access token)", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": 123, "login": true}`, nil}, output{coapCodes.BadRequest, `cannot handle sign in: cbor: cannot unmarshal positive integer`, nil}, true},
		{"Changed1", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}, false},
	}

	for _, test := range tbl {
		tf := func(t *testing.T) {
			co := testCoapDial(t, testCfg.GW_HOST, "")
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

func TestSignInDeviceSubscriptionHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	ctx := kitNetGrpc.CtxWithToken(context.Background(), oauthTest.GetDefaultServiceToken(t))
	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	co.AddOnClose(func() {
		cancel()
	})

	signUpResp := testSignUp(t, CertIdentity, co)
	testSignIn(t, CertIdentity, signUpResp, co)
	_, err = c.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIdFilter: []string{CertIdentity},
	})
	require.NoError(t, err)

	<-cancelCtx.Done()
	require.True(t, cancelCtx.Err() == context.Canceled)

	co1 := testCoapDial(t, testCfg.GW_HOST, "")
	_, code := runSignIn(t, CertIdentity, signUpResp, co1)
	require.Equal(t, coapCodes.Unauthorized, code)
	_ = co1.Close()
}

func TestSignOutPostHandler(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()

	co := testCoapDial(t, testCfg.GW_HOST, "")
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()

	signUpResp := testSignUp(t, CertIdentity, co)
	testSignIn(t, CertIdentity, signUpResp, co)

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

func TestSignInWithMTLSAndDeviceIdClaim(t *testing.T) {
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.APIs.COAP.TLS.Enabled = true
	coapgwCfg.APIs.COAP.TLS.Embedded.ClientCertificateRequired = true
	coapgwCfg.APIs.COAP.Authorization.DeviceIDClaim = oauthService.TokenDeviceID
	shutdown := setUp(t, coapgwCfg)
	defer shutdown()

	signUp := func(deviceID string) service.CoapSignUpResponse {
		co := testCoapDial(t, testCfg.GW_HOST, deviceID)
		require.NotEmpty(t, co)
		signUpResp := testSignUp(t, deviceID, co)
		err := co.Close()
		require.NoError(t, err)
		return signUpResp
	}

	signUpResp := signUp(CertIdentity)
	anotherDeviceID := uuid.New().String()

	check := func(deviceID string, req testEl) {
		co := testCoapDial(t, testCfg.GW_HOST, deviceID)
		require.NotEmpty(t, co)
		testPostHandler(t, uri.SignIn, req, co)
		_ = co.Close()
	}

	tokenWithoutDeviceID := oauthTest.GetDefaultServiceToken(t)

	req := testEl{"OK", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}, false}
	check(CertIdentity, req)

	req = testEl{"mtls deviceID != JWT deviceID", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + signUpResp.AccessToken + `", "login": true }`, nil}, output{coapCodes.Unauthorized, `cannot handle sign in: access token issued to the device`, nil}, true}
	check(anotherDeviceID, req)

	req = testEl{"JWT deviceID is not set", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + signUpResp.UserID + `", "accesstoken":"` + tokenWithoutDeviceID + `", "login": true }`, nil}, output{coapCodes.Unauthorized, `cannot handle sign in: access token doesn't contain the required device id claim`, nil}, true}
	check(CertIdentity, req)
}

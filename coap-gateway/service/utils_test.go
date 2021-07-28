package service_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"

	"github.com/plgd-dev/cloud/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	rdTest "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/kit/security/certManager"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"

	"github.com/kelseyhightower/envconfig"
	authTest "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	raTest "github.com/plgd-dev/cloud/resource-aggregate/test"
	test "github.com/plgd-dev/cloud/test"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type input struct {
	code    codes.Code
	payload interface{}
	queries []string
}

type output input

type testEl struct {
	name string
	in   input
	out  output
}

func testValidateResp(t *testing.T, test testEl, resp *pool.Message) {
	require.Equal(t, test.out.code, resp.Code())
	bodySize, _ := resp.BodySize()
	if bodySize > 0 || test.out.payload != nil {
		body, err := ioutil.ReadAll(resp.Body())
		require.NoError(t, err)
		if contentType, err := resp.ContentFormat(); err == nil {
			switch contentType {
			case message.AppCBOR, message.AppOcfCbor:
				n := reflect.New(reflect.TypeOf(test.out.payload)).Interface()
				err := cbor.Decode(body, n)
				require.NoError(t, err)
				if !assert.Equal(t, test.out.payload, reflect.ValueOf(n).Elem().Interface()) {
					t.Fatal()
				}
			default:
				t.Fatalf("Output payload %v is invalid, expected %v", body, test.out.payload)
			}
		} else {
			// https://tools.ietf.org/html/rfc7252#section-5.5.2
			if v, ok := test.out.payload.(string); ok {
				require.Contains(t, string(body), v)
			} else {
				t.Fatalf("Output payload %v is invalid, expected %v", body, test.out.payload)
			}
		}

		if len(test.out.queries) > 0 {
			queries, err := resp.Options().Queries()
			require.NoError(t, err)
			require.Len(t, queries, len(test.out.queries))
			for idx := range queries {
				if queries[idx] != test.out.queries[idx] {
					t.Fatalf("Invalid query %v, expected %v", queries[idx], test.out.queries[idx])
				}
			}
		}
	}
}

func testSignUp(t *testing.T, deviceID string, co *tcp.ClientConn) service.CoapSignUpResponse {
	code := oauthTest.GetDeviceAuthorizationCode(t)
	signUpReq := service.CoapSignUpRequest{
		DeviceID:              deviceID,
		AuthorizationCode:     code,
		AuthorizationProvider: "plgd",
	}
	inputCbor, err := cbor.Encode(signUpReq)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(req)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(codes.POST)
	req.SetToken(token)
	req.SetPath(uri.SignUp)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	require.NoError(t, err)
	defer pool.ReleaseMessage(resp)

	require.Equal(t, codes.Changed, resp.Code())
	var signUpResp service.CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	require.NoError(t, err)
	require.NotEmpty(t, signUpResp.AccessToken)
	return signUpResp
}

func testSignIn(t *testing.T, r service.CoapSignUpResponse, co *tcp.ClientConn) service.CoapSignInResp {
	signInReq := service.CoapSignInReq{
		DeviceID:    CertIdentity,
		UserID:      r.UserID,
		AccessToken: r.AccessToken,
		Login:       true,
	}
	inputCbor, err := cbor.Encode(signInReq)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := pool.AcquireMessage(ctx)
	defer pool.ReleaseMessage(req)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(codes.POST)
	req.SetToken(token)
	req.SetPath(uri.SignIn)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	require.NoError(t, err)
	defer pool.ReleaseMessage(resp)

	require.Equal(t, codes.Changed, resp.Code())
	var signInResp service.CoapSignInResp
	err = cbor.ReadFrom(resp.Body(), &signInResp)
	require.NoError(t, err)
	return signInResp
}

func testSignUpIn(t *testing.T, deviceID string, co *tcp.ClientConn) service.CoapSignInResp {
	resp := testSignUp(t, deviceID, co)
	return testSignIn(t, resp, co)
}

func testPostHandler(t *testing.T, path string, test testEl, co *tcp.ClientConn) {
	var inputCbor []byte
	var err error
	if v, ok := test.in.payload.(string); ok && v != "" {
		inputCbor, err = json2cbor(v)
	}
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := pool.AcquireMessage(ctx)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(test.in.code)
	req.SetToken(token)
	req.SetPath(path)
	if len(inputCbor) > 0 {
		req.SetContentFormat(message.AppOcfCbor)
		req.SetBody(bytes.NewReader(inputCbor))
	}
	for _, q := range test.in.queries {
		req.AddQuery(q)
	}
	resp, err := co.Do(req)
	require.NoError(t, err)
	testValidateResp(t, test, resp)
}

func json2cbor(data string) ([]byte, error) {
	return json.ToCBOR(data)
}

func testPrepareDevice(t *testing.T, co *tcp.ClientConn) {
	testSignUpIn(t, CertIdentity, co)
	publishResEl := []testEl{
		{"publishResourceA", input{codes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestAResourceHref + `", "rt":["` + TestAResourceType + `"], "type":["` + message.TextPlain.String() + `"] } ], "ttl":12345}`, nil},
			output{codes.Changed, TestWkRD{
				DeviceID:         CertIdentity,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceID:      CertIdentity,
						Href:          TestAResourceHref,
						ResourceTypes: []string{TestAResourceType},
						Type:          []string{message.TextPlain.String()},
					},
				},
			}, nil}},
		{"publishResourceB", input{codes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestBResourceHref + `", "rt":["` + TestBResourceType + `"], "type":["` + message.TextPlain.String() + `"] } ], "ttl":12345}`, nil},
			output{codes.Changed, TestWkRD{
				DeviceID:         CertIdentity,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceID:      CertIdentity,
						Href:          TestBResourceHref,
						ResourceTypes: []string{TestBResourceType},
						Type:          []string{message.TextPlain.String()},
					},
				},
			}, nil}},
	}
	for _, tt := range publishResEl {
		testPostHandler(t, uri.ResourceDirectory, tt, co)
	}
}

func testCoapDial(t *testing.T, host string, withoutTLS ...bool) *tcp.ClientConn {
	var config certManager.OcfConfig
	err := envconfig.Process("LISTEN", &config)
	assert.NoError(t, err)
	listenCertManager, err := certManager.NewOcfCertManager(config)
	require.NoError(t, err)

	tlsConfig := listenCertManager.GetClientTLSConfig()
	tlsConfig.InsecureSkipVerify = true
	tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("empty certificates chain")
		}
		intermediateCAPool := x509.NewCertPool()
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		for _, cert := range certs[1:] {
			intermediateCAPool.AddCert(cert)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{
			Roots:         tlsConfig.RootCAs,
			Intermediates: intermediateCAPool,
			CurrentTime:   time.Now(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err != nil {
			return err
		}

		return nil
	}

	if len(withoutTLS) > 0 {
		tlsConfig = nil
	}
	conn, err := tcp.Dial(host, tcp.WithTLS(tlsConfig), tcp.WithHandlerFunc(func(w *tcp.ResponseWriter, r *pool.Message) {
		switch r.Code() {
		case codes.POST:
			w.SetResponse(codes.Changed, message.TextPlain, bytes.NewReader([]byte("hello world")))
		case codes.GET:
			w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader([]byte("hello world")))
		case codes.PUT:
			w.SetResponse(codes.Created, message.TextPlain, bytes.NewReader([]byte("hello world")))
		case codes.DELETE:
			w.SetResponse(codes.Deleted, message.TextPlain, bytes.NewReader([]byte("hello world")))
		}
	}))
	require.NoError(t, err)
	return conn
}

func setUp(t *testing.T, withoutTLS ...bool) func() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	test.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	auShutdown := authTest.SetUp(t)
	raShutdown := raTest.SetUp(t)
	rdShutdown := rdTest.SetUp(t)
	coapgwCfg := coapgwTest.MakeConfig(t)
	if len(withoutTLS) > 0 {
		coapgwCfg.APIs.COAP.TLS.Enabled = false
	}
	gwShutdown := coapgwTest.New(t, coapgwCfg)
	return func() {
		gwShutdown()
		rdShutdown()
		raShutdown()
		auShutdown()
		oauthShutdown()
	}
}

var (
	AuthorizationUserId       = "1"
	AuthorizationRefreshToken = "refresh-token"

	CertIdentity      = "b5a2a42e-b285-42f1-a36b-034c8fc8efd5"
	TestAResourceHref = "/a"
	TestAResourceId   = (&commands.ResourceId{DeviceId: CertIdentity, Href: TestAResourceHref}).ToUUID()
	TestAResourceType = "x.a"
	TestBResourceHref = "/b"
	TestBResourceId   = (&commands.ResourceId{DeviceId: CertIdentity, Href: TestBResourceHref}).ToUUID()
	TestBResourceType = "x.b"

	TestExchangeTimeout = time.Second * 15
	TestLogDebug        = true
)

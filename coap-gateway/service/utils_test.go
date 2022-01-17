package service_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	idTest "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
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
	name                 string
	in                   input
	out                  output
	allowContextCanceled bool
}

func testValidateResp(t *testing.T, test testEl, resp *pool.Message) {
	require.Equal(t, test.out.code, resp.Code())
	bodySize, _ := resp.BodySize()
	if bodySize == 0 && test.out.payload == nil {
		return
	}
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

func testSignUp(t *testing.T, deviceID string, co *tcp.ClientConn) service.CoapSignUpResponse {
	code := oauthTest.GetDefaultDeviceAuthorizationCode(t, deviceID)
	signUpReq := service.CoapSignUpRequest{
		DeviceID:              deviceID,
		AuthorizationCode:     code,
		AuthorizationProvider: config.DEVICE_PROVIDER,
	}
	inputCbor, err := cbor.Encode(signUpReq)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(codes.POST)
	req.SetToken(token)
	req.SetPath(uri.SignUp)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	require.NoError(t, err)
	defer co.ReleaseMessage(resp)

	require.Equal(t, codes.Changed, resp.Code())
	var signUpResp service.CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	require.NoError(t, err)
	require.NotEmpty(t, signUpResp.AccessToken)
	return signUpResp
}

func doSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *tcp.ClientConn) (*pool.Message, error) {
	signInReq := service.CoapSignInReq{
		DeviceID:    deviceID,
		UserID:      r.UserID,
		AccessToken: r.AccessToken,
		Login:       true,
	}
	inputCbor, err := cbor.Encode(signInReq)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(codes.POST)
	req.SetToken(token)
	req.SetPath(uri.SignIn)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	return co.Do(req)
}

func runSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *tcp.ClientConn) (*service.CoapSignInResp, codes.Code) {
	resp, err := doSignIn(t, deviceID, r, co)
	require.NoError(t, err)
	defer co.ReleaseMessage(resp)

	var signInResp service.CoapSignInResp
	if resp.Code() == codes.Changed {
		err = cbor.ReadFrom(resp.Body(), &signInResp)
		require.NoError(t, err)
		return &signInResp, resp.Code()
	}

	return nil, resp.Code()
}

func testSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *tcp.ClientConn) service.CoapSignInResp {
	signInResp, code := runSignIn(t, deviceID, r, co)
	require.Equal(t, codes.Changed, code)
	require.NotNil(t, signInResp)
	return *signInResp
}

func testSignUpIn(t *testing.T, deviceID string, co *tcp.ClientConn) service.CoapSignInResp {
	resp := testSignUp(t, deviceID, co)
	return testSignIn(t, deviceID, resp, co)
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
	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
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
	if err != nil {
		if errors.Is(err, context.Canceled) && test.allowContextCanceled {
			return
		}
		require.NoError(t, err)
	}
	defer co.ReleaseMessage(resp)
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
			}, nil}, false},
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
			}, nil}, false},
	}
	for _, tt := range publishResEl {
		testPostHandler(t, uri.ResourceDirectory, tt, co)
	}
}

func testCoapDial(t *testing.T, host, deviceID string, withoutTLS ...bool) *tcp.ClientConn {
	var tlsConfig *tls.Config

	if len(withoutTLS) == 0 {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		signerCert, err := security.LoadX509(os.Getenv("TEST_ROOT_CA_CERT"))
		require.NoError(t, err)
		signerKey, err := security.LoadX509PrivateKey(os.Getenv("TEST_ROOT_CA_KEY"))
		require.NoError(t, err)

		certData, err := generateCertificate.GenerateIdentityCert(generateCertificate.Configuration{
			ValidFrom: time.Now().Add(-time.Hour).Format(time.RFC3339),
			ValidFor:  2 * time.Hour,
		}, deviceID, priv, signerCert, signerKey)
		require.NoError(t, err)
		b, err := x509.MarshalECPrivateKey(priv)
		require.NoError(t, err)
		key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
		crt, err := tls.X509KeyPair(certData, key)
		require.NoError(t, err)
		caPool := x509.NewCertPool()
		for _, c := range signerCert {
			caPool.AddCert(c)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{
				crt,
			},
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
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
				caPool := x509.NewCertPool()
				for _, c := range signerCert {
					caPool.AddCert(c)
				}
				_, err := certs[0].Verify(x509.VerifyOptions{
					Roots:         caPool,
					Intermediates: intermediateCAPool,
					CurrentTime:   time.Now(),
					KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				})
				if err != nil {
					return err
				}

				return nil
			},
		}
	}
	conn, err := tcp.Dial(host, tcp.WithTLS(tlsConfig), tcp.WithHandlerFunc(func(w *tcp.ResponseWriter, r *pool.Message) {
		var err error
		resp := []byte("hello world")
		switch r.Code() {
		case codes.POST:
			err = w.SetResponse(codes.Changed, message.TextPlain, bytes.NewReader(resp))
		case codes.GET:
			err = w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader(resp))
		case codes.PUT:
			err = w.SetResponse(codes.Created, message.TextPlain, bytes.NewReader(resp))
		case codes.DELETE:
			err = w.SetResponse(codes.Deleted, message.TextPlain, bytes.NewReader(resp))
		}
		require.NoError(t, err)
	}))
	require.NoError(t, err)
	return conn
}

func setUp(t *testing.T, coapgwCfgs ...service.Config) func() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	testService.ClearDB(ctx, t)
	oauthShutdown := oauthTest.SetUp(t)
	auShutdown := idTest.SetUp(t)
	raShutdown := raTest.SetUp(t)
	rdShutdown := rdTest.SetUp(t)
	grpcShutdown := grpcgwTest.New(t, grpcgwTest.MakeConfig(t))
	coapgwCfg := coapgwTest.MakeConfig(t)
	if len(coapgwCfgs) > 0 {
		coapgwCfg = coapgwCfgs[0]
	}
	gwShutdown := coapgwTest.New(t, coapgwCfg)
	return func() {
		gwShutdown()
		grpcShutdown()
		rdShutdown()
		raShutdown()
		auShutdown()
		oauthShutdown()
	}
}

var (
	AuthorizationUserID       = "1"
	AuthorizationRefreshToken = oauthTest.ValidRefreshToken

	CertIdentity      = "b5a2a42e-b285-42f1-a36b-034c8fc8efd5"
	TestAResourceHref = "/a"
	TestAResourceID   = (&commands.ResourceId{DeviceId: CertIdentity, Href: TestAResourceHref}).ToUUID()
	TestAResourceType = "x.a"
	TestBResourceHref = "/b"
	TestBResourceID   = (&commands.ResourceId{DeviceId: CertIdentity, Href: TestBResourceHref}).ToUUID()
	TestBResourceType = "x.b"

	TestExchangeTimeout = time.Second * 15
	TestLogDebug        = true
)

//go:build test
// +build test

package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/test"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
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
	body, err := io.ReadAll(resp.Body())
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

func doSignUp(t *testing.T, deviceID string, co *coapTcpClient.Conn) (*pool.Message, error) {
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
	err = req.SetPath(uri.SignUp)
	require.NoError(t, err)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	return co.Do(req)
}

func testSignUp(t *testing.T, deviceID string, co *coapTcpClient.Conn) service.CoapSignUpResponse {
	resp, err := doSignUp(t, deviceID, co)
	require.NoError(t, err)
	defer co.ReleaseMessage(resp)

	require.Equal(t, codes.Changed, resp.Code())
	var signUpResp service.CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	require.NoError(t, err)
	require.NotEmpty(t, signUpResp.AccessToken)
	return signUpResp
}

func doSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *coapTcpClient.Conn) (*pool.Message, error) {
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
	err = req.SetPath(uri.SignIn)
	require.NoError(t, err)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	return co.Do(req)
}

func runSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *coapTcpClient.Conn) (*service.CoapSignInResp, codes.Code) {
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

func testSignIn(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *coapTcpClient.Conn) {
	signInResp, code := runSignIn(t, deviceID, r, co)
	require.Equal(t, codes.Changed, code)
	require.NotNil(t, signInResp)
}

func testSignUpIn(t *testing.T, deviceID string, co *coapTcpClient.Conn) {
	resp := testSignUp(t, deviceID, co)
	testSignIn(t, deviceID, resp, co)
}

func testRefreshTokenWithResp(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *coapTcpClient.Conn) *pool.Message {
	refreshTokenReq := service.CoapRefreshTokenReq{
		DeviceID:              deviceID,
		UserID:                r.UserID,
		RefreshToken:          r.RefreshToken,
		AuthorizationProvider: config.DEVICE_PROVIDER,
	}
	inputCbor, err := cbor.Encode(refreshTokenReq)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(co.Context(), TestExchangeTimeout)
	defer cancel()
	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	require.NoError(t, err)
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(uri.RefreshToken)
	require.NoError(t, err)
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	require.NoError(t, err)
	return resp
}

func testRefreshToken(t *testing.T, deviceID string, r service.CoapSignUpResponse, co *coapTcpClient.Conn) service.CoapRefreshTokenResp {
	resp := testRefreshTokenWithResp(t, deviceID, r, co)
	require.Equal(t, codes.Changed, resp.Code())
	var refreshTokenResp service.CoapRefreshTokenResp
	err := cbor.ReadFrom(resp.Body(), &refreshTokenResp)
	require.NoError(t, err)
	require.NotEmpty(t, refreshTokenResp.AccessToken)
	return refreshTokenResp
}

func testPostHandler(t *testing.T, path string, test testEl, co *coapTcpClient.Conn) {
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
	err = req.SetPath(path)
	require.NoError(t, err)
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

func testPublishResources(t *testing.T, deviceID string, co *coapTcpClient.Conn) {
	publishResEl := []testEl{
		{
			"publishResourceA",
			input{codes.POST, `{ "di":"` + deviceID + `", "links":[ { "di":"` + deviceID + `", "href":"` + TestAResourceHref + `", "rt":["` + TestAResourceType + `"], "type":["` + message.TextPlain.String() + `"], "p":{"bm":3} } ], "ttl":12345}`, nil},
			output{codes.Changed, TestWkRD{
				DeviceID:         deviceID,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceID:      deviceID,
						Href:          TestAResourceHref,
						ResourceTypes: []string{TestAResourceType},
						Type:          []string{message.TextPlain.String()},
					},
				},
			}, nil},
			false,
		},
		{
			"publishResourceB",
			input{codes.POST, `{ "di":"` + deviceID + `", "links":[ { "di":"` + deviceID + `", "href":"` + TestBResourceHref + `", "rt":["` + TestBResourceType + `"], "type":["` + message.TextPlain.String() + `"], "p":{"bm":3} } ], "ttl":12345}`, nil},
			output{codes.Changed, TestWkRD{
				DeviceID:         deviceID,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceID:      deviceID,
						Href:          TestBResourceHref,
						ResourceTypes: []string{TestBResourceType},
						Type:          []string{message.TextPlain.String()},
					},
				},
			}, nil},
			false,
		},
	}
	for _, tt := range publishResEl {
		testPostHandler(t, uri.ResourceDirectory, tt, co)
	}
	time.Sleep(time.Second) // for publish content of device resources
}

func testPrepareDevice(t *testing.T, co *coapTcpClient.Conn) {
	testSignUpIn(t, CertIdentity, co)
	testPublishResources(t, CertIdentity, co)
}

func handleDiscoveryResource(t *testing.T, w *responsewriter.ResponseWriter[*coapTcpClient.Conn], r *pool.Message) {
	links := schema.ResourceLinks{
		{
			Href:          resources.ResourceURI,
			ResourceTypes: []string{resources.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_LL},
			DeviceID:      CertIdentity,
			Policy: &schema.Policy{
				BitMask: schema.Discoverable,
			},
		},
		{
			Href:          TestAResourceHref,
			ResourceTypes: []string{TestAResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
			DeviceID:      CertIdentity,
			Policy: &schema.Policy{
				BitMask: schema.Discoverable | schema.Observable,
			},
		},
		{
			Href:          TestBResourceHref,
			ResourceTypes: []string{TestAResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE},
			DeviceID:      CertIdentity,
			Policy: &schema.Policy{
				BitMask: schema.Discoverable | schema.Observable,
			},
		},
	}
	data, err := cbor.Encode(links)
	require.NoError(t, err)
	err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader(data))
	require.NoError(t, err)
}

func makeTestCoapHandler(t *testing.T) func(w *responsewriter.ResponseWriter[*coapTcpClient.Conn], r *pool.Message) {
	return func(w *responsewriter.ResponseWriter[*coapTcpClient.Conn], r *pool.Message) {
		var err error
		resp := []byte("hello world")
		switch r.Code() {
		case codes.POST:
			err = w.SetResponse(codes.Changed, message.TextPlain, bytes.NewReader(resp))
		case codes.GET:
			path, err := r.Path()
			if err == nil && path == uri.ResourceDiscovery {
				handleDiscoveryResource(t, w, r)
				return
			}
			respOptions := message.Options{
				message.Option{ID: message.ETag, Value: []byte(TestETag)},
			}
			_, err = r.Options().Observe()
			if err == nil {
				respOptions, _, _ = respOptions.AddUint32(make([]byte, 10), message.Observe, 12345)
			}
			etag, err := r.ETag()
			if err == nil && bytes.Equal(etag, []byte(TestETag)) {
				err = w.SetResponse(codes.Valid, message.TextPlain, nil, respOptions...)
			} else {
				err = w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader(resp), respOptions...)
			}
		case codes.PUT:
			err = w.SetResponse(codes.Created, message.TextPlain, bytes.NewReader(resp))
		case codes.DELETE:
			err = w.SetResponse(codes.Deleted, message.TextPlain, bytes.NewReader(resp))
		}
		require.NoError(t, err)
	}
}

type testCoapDialConfig struct {
	generateTLS *struct {
		deviceID     string
		identityCert bool
		validTo      time.Time
	}
	tlsConfig *tls.Config
}

type option interface {
	apply(*testCoapDialConfig)
}

type optionFunc func(*testCoapDialConfig)

func (o optionFunc) apply(c *testCoapDialConfig) {
	o(c)
}

func WithGenerateTLS(deviceID string, identityCert bool, validTo time.Time) option {
	return optionFunc(func(cfg *testCoapDialConfig) {
		var generateTLS struct {
			deviceID     string
			identityCert bool
			validTo      time.Time
		}
		generateTLS.deviceID = deviceID
		generateTLS.identityCert = identityCert
		generateTLS.validTo = validTo
		cfg.generateTLS = &generateTLS
	})
}

func WithTLSConfig(tlsConfig *tls.Config) option {
	return optionFunc(func(cfg *testCoapDialConfig) {
		cfg.tlsConfig = tlsConfig
	})
}

func testCoapDialWithHandler(t *testing.T, h func(w *responsewriter.ResponseWriter[*coapTcpClient.Conn], r *pool.Message), opts ...option) *coapTcpClient.Conn {
	c := &testCoapDialConfig{}
	for _, opt := range opts {
		opt.apply(c)
	}
	tlsConfig := c.tlsConfig
	if c.generateTLS != nil {
		signerCert, err := pkgX509.ReadX509(os.Getenv("TEST_ROOT_CA_CERT"))
		require.NoError(t, err)
		signerKey, err := pkgX509.ReadPrivateKey(os.Getenv("TEST_ROOT_CA_KEY"))
		require.NoError(t, err)
		cg := test.NewLocalCertificateGenerator(signerCert, signerKey)
		var crt tls.Certificate
		if c.generateTLS.identityCert {
			crt, err = cg.GetIdentityCertificate(c.generateTLS.deviceID, c.generateTLS.validTo)
		} else {
			crt, err = cg.GetCertificate(c.generateTLS.validTo)
		}
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
					return errors.New("empty certificates chain")
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
					Roots:         caPool,
					Intermediates: intermediateCAPool,
					CurrentTime:   time.Now(),
					KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				})
				return err
			},
		}
	}
	conn, err := tcp.Dial(config.COAP_GW_HOST, options.WithTLS(tlsConfig), options.WithHandlerFunc(h))
	require.NoError(t, err)
	return conn
}

func testCoapDial(t *testing.T, deviceID string, withTLS, identityCert bool, validTo time.Time) *coapTcpClient.Conn {
	var opts []option
	if withTLS {
		opts = append(opts, WithGenerateTLS(deviceID, identityCert, validTo))
	}
	return testCoapDialWithHandler(t, makeTestCoapHandler(t), opts...)
}

func setUp(t *testing.T, coapgwCfgs ...service.Config) func() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	testService.ClearDB(ctx, t)
	coapgwCfg := coapgwTest.MakeConfig(t)
	if len(coapgwCfgs) > 0 {
		coapgwCfg = coapgwCfgs[0]
	}
	return testService.SetUpServices(context.Background(), t, testService.SetUpServicesMachine2MachineOAuth|testService.SetUpServicesCertificateAuthority|testService.SetUpServicesOAuth|
		testService.SetUpServicesId|testService.SetUpServicesResourceAggregate|testService.SetUpServicesResourceDirectory|testService.SetUpServicesCoapGateway|testService.SetUpServicesGrpcGateway, testService.WithCOAPGWConfig(coapgwCfg))
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
	TestETag            = "12345678"
)

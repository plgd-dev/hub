package service

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/uri"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/net/coap"

	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/kit/security/certManager/acme/ocf"

	coapCodes "github.com/go-ocf/go-coap/codes"
	kitNetCoap "github.com/go-ocf/kit/net/coap"

	oauthTest "github.com/go-ocf/cloud/authorization/provider"
	authConfig "github.com/go-ocf/cloud/authorization/service"
	authService "github.com/go-ocf/cloud/authorization/test/service"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	refImplRA "github.com/go-ocf/cloud/resource-aggregate/refImpl"
	raService "github.com/go-ocf/cloud/resource-aggregate/test/service"
	refImplRD "github.com/go-ocf/cloud/resource-directory/refImpl"
	rdService "github.com/go-ocf/cloud/resource-directory/test/service"
	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/log"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type input struct {
	code    coapCodes.Code
	payload interface{}
	queries []string
}

type output input

type testEl struct {
	name string
	in   input
	out  output
}

func testCreateResourceStoreSub(t *testing.T, resourceDBname string) (*mongodb.EventStore, *nats.Subscriber) {
	var natsCfg nats.Config
	err := envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	mgoCfg.DatabaseName = resourceDBname
	assert.NoError(t, err)

	var cmconfig certManager.Config
	err = envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()

	subscriber, err := nats.NewSubscriber(natsCfg, nil, func(err error) { log.Errorf("%v", err) }, nats.WithTLS(&tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(&tlsConfig))
	assert.NoError(t, err)
	return eventstore, subscriber
}

func initializeStruct(t reflect.Type, v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		switch ft.Type.Kind() {
		case reflect.Map:
			f.Set(reflect.MakeMap(ft.Type))
		case reflect.Slice:
			f.Set(reflect.MakeSlice(ft.Type, 0, 0))
		case reflect.Chan:
			f.Set(reflect.MakeChan(ft.Type, 0))
		case reflect.Struct:
			initializeStruct(ft.Type, f)
		case reflect.Ptr:
			fv := reflect.New(ft.Type.Elem())
			initializeStruct(ft.Type.Elem(), fv.Elem())
			f.Set(fv)
		default:
		}
	}
}

func testValidateResp(t *testing.T, test testEl, resp gocoap.Message) {
	if resp.Code() != test.out.code {
		t.Fatalf("Output code %v is invalid, expected %v", resp.Code(), test.out.code)
	} else {
		if len(resp.Payload()) > 0 || test.out.payload != nil {
			if contentType, ok := resp.Option(gocoap.ContentFormat).(gocoap.MediaType); ok {
				switch contentType {
				case gocoap.AppCBOR, gocoap.AppOcfCbor:
					n := reflect.New(reflect.TypeOf(test.out.payload)).Interface()
					err := cbor.Decode(resp.Payload(), n)
					if err != nil {
						t.Fatalf("Cannot convert cbor to type: %v %v", err, n)
					}
					if !assert.Equal(t, test.out.payload, reflect.ValueOf(n).Elem().Interface()) {
						t.Fatal()
					}
				case gocoap.TextPlain:
					if v, ok := test.out.payload.(string); ok {
						if strings.Count(string(resp.Payload()), v) == 0 {
							t.Fatalf("Output payload '%v' is invalid, expected '%v'", string(resp.Payload()), test.out.payload)
						}
					} else {
						t.Fatalf("Output payload %v is invalid, expected %v", resp.Payload(), test.out.payload)
					}
				}
			} else {
				t.Fatalf("Output payload %v is invalid, expected %v", resp.Payload(), test.out.payload)
			}
		}
		if len(test.out.queries) > 0 {
			queries := resp.Options(gocoap.URIQuery)
			if resp == nil {
				t.Fatalf("Output doesn't contains queries, expected: %v", test.out.queries)
			}
			if len(queries) == len(test.out.queries) {
				t.Fatalf("Invalid queries %v, expected: %v", queries, test.out.queries)
			}
			for idx := range queries {
				if queries[idx] != test.out.queries[idx] {
					t.Fatalf("Invalid query %v, expected %v", queries[idx], test.out.queries[idx])
				}
			}
		}
	}
}

func testPostHandler(t *testing.T, path string, test testEl, co *gocoap.ClientConn) {
	var inputCbor []byte
	var err error
	if v, ok := test.in.payload.(string); ok && v != "" {
		inputCbor, err = json2cbor(v)
	}
	if err != nil {
		t.Fatalf("Cannot convert json to cbor: %v", err)
	}

	req := co.NewMessage(gocoap.MessageParams{
		Code: test.in.code,
		Token: func() []byte {
			token, err := gocoap.GenerateToken()
			if err != nil {
				t.Fatalf("Cannot generate token: %v", err)
			}
			return token
		}(),
		MessageID: gocoap.GenerateMessageID(),
	})
	req.SetPathString(path)
	if len(inputCbor) > 0 {
		req.AddOption(gocoap.ContentFormat, gocoap.AppOcfCbor)
		req.SetPayload(inputCbor)
	}
	for _, q := range test.in.queries {
		req.AddOption(gocoap.URIQuery, q)
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestExchangeTimeout)
	defer cancel()
	resp, err := co.ExchangeWithContext(ctx, req)
	if err != nil {
		t.Fatalf("Cannot send/retrieve msg: %v", err)
	}
	testValidateResp(t, test, resp)
}

func json2cbor(data string) ([]byte, error) {
	return json.ToCBOR(data)
}

func cannonalizeJSON(data string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	var m interface{}
	err := json.Decode([]byte(data), &m)
	if err != nil {
		return "", err
	}
	out, err := json.Encode(m)
	return string(out), err
}

func cbor2json(data []byte) (string, error) {
	return cbor.ToJSON(data)
}

func testCreateCoapGateway(t *testing.T, resourceDBname string, config Config) func() {
	eventstore, subscriber := testCreateResourceStoreSub(t, resourceDBname)
	var acmeCfg certManager.Config
	err := envconfig.Process("DIAL", &acmeCfg)
	assert.NoError(t, err)

	clientCertManager, err := certManager.NewCertManager(acmeCfg)
	require.NoError(t, err)

	var listenCertManager ListenCertManager
	if strings.HasSuffix(config.Net, "-tls") {
		var acmeOcfCfg ocf.Config
		err = envconfig.Process("LISTEN_ACME", &acmeOcfCfg)
		assert.NoError(t, err)
		coapGWAcmeDirectory := os.Getenv("TEST_COAP_GW_OVERWRITE_LISTEN_ACME_DIRECTORY_URL")
		require.NotEmpty(t, coapGWAcmeDirectory)
		acmeOcfCfg.CADirURL = coapGWAcmeDirectory
		listenCertManager, err = ocf.NewAcmeCertManagerFromConfiguration(acmeOcfCfg)
		require.NoError(t, err)
	}

	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	server := New(config, clientCertManager, listenCertManager, func(ctx context.Context, code coapCodes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SecureRefreshToken, uri.SignUp, uri.SecureSignUp, uri.SignIn, uri.SecureSignIn, uri.ResourcePing:
			return ctx, nil
		}
		_, err := kitNetCoap.TokenFromCtx(ctx)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}, eventstore, subscriber, pool)
	server.setupCoapServer()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Serve()
	}()

	return func() {
		server.Shutdown()
		wg.Wait()
	}
}

func testCreateResourceAggregate(t *testing.T, resourceDBname, addr, AuthServerAddr string) (shutdown func()) {
	var config refImplRA.Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.Service.AuthServerAddr = AuthServerAddr
	config.MongoDB.DatabaseName = resourceDBname
	config.Service.Addr = addr
	//config.Log.Debug = TestLogDebug
	config.Service.SnapshotThreshold = 1

	return raService.NewResourceAggregate(t, config)
}

func init() {
	log.Setup(log.Config{Debug: TestLogDebug})
}

func testPrepareDevice(t *testing.T, co *gocoap.ClientConn) {

	signUpEl := testEl{"signUp", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "accesstoken": "123", "authprovider": "` + oauthTest.NewTestProvider().GetProviderName() + `"}`, nil}, output{coapCodes.Changed, TestCoapSignUpResponse{RefreshToken: "refresh-token", UserId: AuthorizationUserId}, nil}}
	testPostHandler(t, uri.SignUp, signUpEl, co)
	signInEl := testEl{"signIn", input{coapCodes.POST, `{"di": "` + CertIdentity + `", "uid":"` + AuthorizationUserId + `", "accesstoken":"` + oauthTest.UserToken + `", "login": true }`, nil}, output{coapCodes.Changed, TestCoapSignInResponse{}, nil}}
	testPostHandler(t, uri.SignIn, signInEl, co)
	publishResEl := []testEl{
		{"publishResourceA", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestAResourceHref + `", "rt":["` + TestAResourceType + `"], "type":["` + gocoap.TextPlain.String() + `"] } ], "ttl":12345}`, nil},
			output{coapCodes.Changed, TestWkRD{
				DeviceID:         CertIdentity,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceId:      CertIdentity,
						Href:          TestAResourceHref,
						Id:            TestAResourceId,
						ResourceTypes: []string{TestAResourceType},
						Type:          []string{gocoap.TextPlain.String()},
					},
				},
			}, nil}},
		{"publishResourceB", input{coapCodes.POST, `{ "di":"` + CertIdentity + `", "links":[ { "di":"` + CertIdentity + `", "href":"` + TestBResourceHref + `", "rt":["` + TestBResourceType + `"], "type":["` + gocoap.TextPlain.String() + `"] } ], "ttl":12345}`, nil},
			output{coapCodes.Changed, TestWkRD{
				DeviceID:         CertIdentity,
				TimeToLive:       12345,
				TimeToLiveLegacy: 12345,
				Links: []TestResource{
					{
						DeviceId:      CertIdentity,
						Href:          TestBResourceHref,
						Id:            TestBResourceId,
						ResourceTypes: []string{TestBResourceType},
						Type:          []string{gocoap.TextPlain.String()},
					},
				},
			}, nil}},
	}
	for _, tt := range publishResEl {
		testPostHandler(t, uri.ResourceDirectory, tt, co)
	}
}

func testCreateResourceDirectory(t *testing.T, resourceDBname, addr, AuthServerAddr string) func() {
	var config refImplRD.Config
	err := envconfig.Process("", &config)
	assert.NoError(t, err)
	config.Service.AuthServerAddr = AuthServerAddr
	config.MongoDB.DatabaseName = resourceDBname
	config.Service.Addr = addr
	//config.Log.Debug = TestLogDebug

	return rdService.NewResourceDirectory(t, config)
}

func testCreateAuthServer(t *testing.T, addr string) func() {
	var authConfig authConfig.Config

	envconfig.Process("", &authConfig)
	var acmeCfg certManager.Config
	err := envconfig.Process("DIAL", &acmeCfg)
	assert.NoError(t, err)
	authConfig.Listen = acmeCfg
	require.NoError(t, err)
	authConfig.Addr = addr

	return authService.NewAuthServer(t, authConfig)
}

func testCoapDial(t *testing.T, host, net string) *gocoap.ClientConn {
	var config certManager.OcfConfig
	err := envconfig.Process("LISTEN", &config)
	assert.NoError(t, err)
	config.Acme.DeviceID = CertIdentity

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
		if coap.VerifyIndetityCertificate(certs[0]) != nil {
			return err
		}
		return nil
	}

	c := &gocoap.Client{Net: net, TLSConfig: &tlsConfig, Handler: func(w gocoap.ResponseWriter, req *gocoap.Request) {
		switch req.Msg.Code() {
		case coapCodes.POST, coapCodes.GET, coapCodes.PUT, coapCodes.DELETE:
			w.SetContentFormat(gocoap.TextPlain)
			w.Write([]byte("hello world"))
		}
	}}
	conn, err := c.Dial(host)
	assert.NoError(t, err)
	return conn
}

/*
type mockResponseWriter struct {
	gocoap.ResponseWriter
	code coapCodes.Code
}

func (m *mockResponseWriter) NewResponse(code coapCodes.Code) gocoap.Message   { m.code = code; return nil }
func (m *mockResponseWriter) Write(p []byte) (n int, err error)             { return -1, nil }
func (m *mockResponseWriter) SetCode(code coapCodes.Code)                    { m.code = code }
func (m *mockResponseWriter) SetContentFormat(contentFormat gocoap.MediaType) {}
func (m *mockResponseWriter) WriteMsg(gocoap.Message) error                   { return nil }
*/

var (
	AuthorizationUserId       = "1"
	AuthorizationRefreshToken = "refresh-token"

	CertIdentity      = "b5a2a42e-b285-42f1-a36b-034c8fc8efd5"
	TestAResourceHref = "/a"
	TestAResourceId   = resource2UUID(CertIdentity, TestAResourceHref)
	TestAResourceType = "x.a"
	TestBResourceHref = "/b"
	TestBResourceId   = resource2UUID(CertIdentity, TestBResourceHref)
	TestBResourceType = "x.b"

	TestExchangeTimeout = time.Second * 15
	TestLogDebug        = true
)

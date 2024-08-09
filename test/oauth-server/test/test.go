package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

const (
	ClientTest = "test"
	// Client with short auth code and access token expiration
	ClientTestShortExpiration = "testShortExpiration"
	// Client will return error when the same auth code or refresh token
	// is used repeatedly within a minute of the first use
	ClientTestRestrictedAuth = "testRestrictedAuth"
	// Client with expired access token
	ClientTestExpired = "testExpired"
	// Client for C2C testing
	ClientTestC2C = "testC2C"
	// Client with configured required params used to verify the authorization and token request query params
	ClientTestRequiredParams = "requiredParams"
	// Secret for client with configured required params
	ClientTestRequiredParamsSecret = "requiredParamsSecret"
	// Valid refresh token if refresh token restriction policy not configured
	ValidRefreshToken = "refresh-token"
)

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.OAUTH_SERVER_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	cfg.Clients.OpenTelemetryCollector = kitNetHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}

	cfg.OAuthSigner.IDTokenKeyFile = urischeme.URIScheme(os.Getenv("TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY"))
	cfg.OAuthSigner.AccessTokenKeyFile = urischeme.URIScheme(os.Getenv("TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY"))
	cfg.OAuthSigner.Domain = config.OAUTH_SERVER_HOST
	cfg.OAuthSigner.Clients = service.OAuthClientsConfig{
		{
			ID:                              config.OAUTH_MANAGER_CLIENT_ID,
			AuthorizationCodeLifetime:       time.Minute * 10,
			AccessTokenLifetime:             0,
			CodeRestrictionLifetime:         0,
			RefreshTokenRestrictionLifetime: 0,
		},
		{
			ID:                              ClientTestShortExpiration,
			AuthorizationCodeLifetime:       time.Second * 10,
			AccessTokenLifetime:             time.Second * 10,
			CodeRestrictionLifetime:         0,
			RefreshTokenRestrictionLifetime: 0,
		},
		{
			ID:                              ClientTestRestrictedAuth,
			AuthorizationCodeLifetime:       time.Minute * 10,
			AccessTokenLifetime:             time.Hour * 24,
			CodeRestrictionLifetime:         time.Minute,
			RefreshTokenRestrictionLifetime: time.Minute,
		},
		{
			ID:                              ClientTestExpired,
			AuthorizationCodeLifetime:       time.Second * 10,
			AccessTokenLifetime:             -1 * time.Second,
			CodeRestrictionLifetime:         0,
			RefreshTokenRestrictionLifetime: 0,
		},
		{
			ID:                              ClientTestC2C,
			AuthorizationCodeLifetime:       time.Minute * 10,
			AccessTokenLifetime:             0,
			CodeRestrictionLifetime:         0,
			RefreshTokenRestrictionLifetime: 0,
			ConsentScreenEnabled:            true,
		},
		{
			ID:                              ClientTestRequiredParams,
			ClientSecret:                    ClientTestRequiredParamsSecret,
			AuthorizationCodeLifetime:       time.Minute * 10,
			AccessTokenLifetime:             time.Minute * 10,
			CodeRestrictionLifetime:         time.Minute * 10,
			RefreshTokenRestrictionLifetime: time.Minute * 10,
			ConsentScreenEnabled:            false,
			RequireIssuedAuthorizationCode:  true,
			RequiredScope:                   []string{"offline_access", "r:*"},
			RequiredResponseType:            "code",
			RequiredRedirectURI:             "http://localhost:7777",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func SetUp(t require.TestingT) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t require.TestingT, cfg service.Config) func() {
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()
	return func() {
		_ = s.Close()
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

func NewRequestBuilder(method, host, url string, body io.Reader) *RequestBuilder {
	b := RequestBuilder{
		method:      method,
		body:        body,
		uri:         fmt.Sprintf("https://%s%s", host, url),
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string]string),
	}
	return &b
}

type RequestBuilder struct {
	method      string
	body        io.Reader
	uri         string
	uriParams   map[string]interface{}
	header      map[string]string
	queryParams map[string]string
}

func (c *RequestBuilder) AddQuery(key, value string) *RequestBuilder {
	c.queryParams[key] = value
	return c
}

func (c *RequestBuilder) Build() *http.Request {
	tmp, _ := uritemplates.Parse(c.uri)
	uri, _ := tmp.Expand(c.uriParams)
	url, _ := url.Parse(uri)
	query := url.Query()
	for k, v := range c.queryParams {
		query.Set(k, v)
	}
	url.RawQuery = query.Encode()
	request, _ := http.NewRequest(c.method, url.String(), c.body)
	for k, v := range c.header {
		request.Header.Add(k, v)
	}
	return request
}

func HTTPDo(t require.TestingT, req *http.Request, followRedirect bool) *http.Response {
	trans := http.DefaultTransport.(*http.Transport).Clone()
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	c := &http.Client{
		Transport: trans,
	}
	if !followRedirect {
		c.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

func GetAccessToken(t require.TestingT, authServerHost, clientID string, claimOverrides map[string]interface{}) string {
	code := GetAuthorizationCode(t, authServerHost, clientID, "", "r:* w:*")
	reqBody := map[string]interface{}{
		uri.GrantTypeKey: string(service.AllowedGrantType_AUTHORIZATION_CODE),
		uri.ClientIDKey:  clientID,
		uri.CodeKey:      code,
	}
	if claimOverrides != nil {
		reqBody[uri.ClaimOverridesKey] = claimOverrides
	}
	d, err := json.Encode(reqBody)
	require.NoError(t, err)

	getReq := NewRequestBuilder(http.MethodPost, authServerHost, uri.Token, bytes.NewReader(d)).Build()
	res := HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var body map[string]string
	err = json.ReadFrom(res.Body, &body)
	require.NoError(t, err)
	token := body["access_token"]
	require.NotEmpty(t, token)
	return token
}

func GetDefaultAccessToken(t require.TestingT) string {
	return GetAccessToken(t, config.OAUTH_SERVER_HOST, ClientTest, nil)
}

func GetJWTValidator(jwkURL string) *jwt.Validator {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	client := http.Client{
		Transport: t,
		Timeout:   time.Second * 10,
	}
	return jwt.NewValidator(jwt.NewKeyCache(jwkURL, &client), log.Get())
}

func GetAuthorizationCode(t require.TestingT, authServerHost, clientID, deviceID, scopes string) string {
	u, err := url.Parse(uri.Authorize)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	q.Add(uri.ClientIDKey, clientID)
	if deviceID != "" {
		q.Add(uri.DeviceIDKey, deviceID)
	}
	if scopes != "" {
		q.Add(uri.ScopeKey, scopes)
	}
	u.RawQuery = q.Encode()
	getReq := NewRequestBuilder(http.MethodGet, authServerHost, u.String(), nil).Build()
	res := HTTPDo(t, getReq, false)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, http.StatusOK, res.StatusCode)

	var body map[string]string
	err = json.ReadFrom(res.Body, &body)
	require.NoError(t, err)
	code := body[uri.CodeKey]
	require.NotEmpty(t, code)
	return code
}

func GetDefaultDeviceAuthorizationCode(t require.TestingT, deviceID string) string {
	return GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, ClientTest, deviceID, "")
}

package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	testOAuthUri "github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

const (
	OwnerClaim    = testOAuthUri.OwnerClaimKey
	DeviceIDClaim = testOAuthUri.DeviceIDClaimKey
)

var ServiceOAuthClient = service.Client{
	ID:                  "serviceClient",
	SecretFile:          "data:,serviceClientSecret",
	RequireDeviceID:     false,
	RequireOwner:        false,
	AccessTokenLifetime: 0,
	AllowedGrantTypes:   []service.GrantType{service.GrantTypeClientCredentials},
	AllowedAudiences:    nil,
	AllowedScopes:       nil,
}

var JWTPrivateKeyOAuthClient = service.Client{
	ID:                  "JWTPrivateKeyClient",
	SecretFile:          "data:,JWTPrivateKeyClientSecret",
	RequireDeviceID:     false,
	RequireOwner:        true,
	AccessTokenLifetime: 0,
	AllowedGrantTypes:   []service.GrantType{service.GrantTypeClientCredentials},
	AllowedAudiences:    nil,
	AllowedScopes:       nil,
	JWTPrivateKey: service.PrivateKeyJWTConfig{
		Enabled:       true,
		Authorization: config.MakeAuthorizationConfig(),
	},
}

var DeviceProvisioningServiceOAuthClient = service.Client{
	ID:                  "deviceProvisioningServiceClient",
	SecretFile:          "data:,deviceProvisioningServiceClientSecret",
	RequireDeviceID:     true,
	RequireOwner:        true,
	AccessTokenLifetime: 0,
	AllowedGrantTypes:   []service.GrantType{service.GrantTypeClientCredentials},
	AllowedAudiences:    nil,
	AllowedScopes:       nil,
}

var OAuthClients = service.OAuthClientsConfig{
	&ServiceOAuthClient,
	&JWTPrivateKeyOAuthClient,
	&DeviceProvisioningServiceOAuthClient,
}

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.M2M_OAUTH_SERVER_HTTP_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	cfg.Clients.OpenTelemetryCollector = kitNetHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}

	cfg.OAuthSigner.PrivateKeyFile = urischeme.URIScheme(os.Getenv("M2M_OAUTH_SERVER_PRIVATE_KEY"))
	cfg.OAuthSigner.Domain = config.M2M_OAUTH_SERVER_HTTP_HOST
	cfg.OAuthSigner.Clients = OAuthClients
	cfg.OAuthSigner.OwnerClaim = OwnerClaim
	cfg.OAuthSigner.DeviceIDClaim = DeviceIDClaim
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

func GetSecret(t require.TestingT, clientID string) string {
	for _, c := range OAuthClients {
		if c.ID == clientID {
			data, err := c.SecretFile.Read()
			require.NoError(t, err)
			return string(data)
		}
	}
	require.FailNow(t, "client not found")
	return ""
}

/*

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
*/

type AccessTokenOptions struct {
	Host         string
	ClientID     string
	ClientSecret string
	Owner        string
	DeviceID     string
	GrantType    string
	Audience     string
	JWT          string
	PostForm     bool
	Ctx          context.Context
}

func WithAccessTokenOptions(options AccessTokenOptions) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		*opts = options
	}
}

func WithAccessTokenClientID(clientID string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.ClientID = clientID
	}
}

func WithAccessTokenClientSecret(clientSecret string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.ClientSecret = clientSecret
	}
}

func WithAccessTokenOwner(owner string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.Owner = owner
	}
}

func WithAccessTokenDeviceID(deviceID string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.DeviceID = deviceID
	}
}

func WithAccessTokenGrantType(grantType string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.GrantType = grantType
	}
}

func WithAccessTokenAudience(audience string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.Audience = audience
	}
}

func WithAccessTokenHost(host string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.Host = host
	}
}

func WithAccessTokenJWT(jwt string) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.JWT = jwt
	}
}

func WithPostFrom(enabled bool) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.PostForm = enabled
	}
}

func WithContext(ctx context.Context) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.Ctx = ctx
	}
}

func mapToURLValues(data map[string]interface{}) url.Values {
	values := url.Values{}
	for key, value := range data {
		values.Add(key, strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%v", value)), " ", "%20")))
	}
	return values
}

func GetAccessToken(t *testing.T, expectedCode int, opts ...func(opts *AccessTokenOptions)) map[string]string {
	options := AccessTokenOptions{
		Host:         config.M2M_OAUTH_SERVER_HTTP_HOST,
		ClientID:     ServiceOAuthClient.ID,
		ClientSecret: GetSecret(t, ServiceOAuthClient.ID),
		GrantType:    string(service.GrantTypeClientCredentials),
		Ctx:          context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}
	reqBody := map[string]interface{}{
		uri.GrantTypeKey: options.GrantType,
		uri.ClientIDKey:  options.ClientID,
	}

	if options.Owner != "" {
		reqBody[uri.OwnerKey] = options.Owner
	}
	if options.DeviceID != "" {
		reqBody[uri.DeviceIDKey] = options.DeviceID
	}
	if options.Audience != "" {
		reqBody[uri.AudienceKey] = options.Audience
	}
	if options.JWT != "" {
		reqBody[uri.ClientAssertionKey] = options.JWT
		reqBody[uri.ClientAssertionTypeKey] = uri.ClientAssertionTypeJWT
	}
	var data []byte
	if options.PostForm {
		data = []byte(mapToURLValues(reqBody).Encode())
	} else {
		var err error
		data, err = json.Encode(reqBody)
		require.NoError(t, err)
	}

	req := testHttp.NewRequest(http.MethodPost, testHttp.HTTPS_SCHEME+options.Host+uri.Token, bytes.NewReader(data)).Build(options.Ctx, t)
	require.NotNil(t, req)
	req.SetBasicAuth(options.ClientID, options.ClientSecret)
	if options.PostForm {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res := testHttp.Do(t, req)
	defer func() {
		_ = res.Body.Close()
	}()
	require.Equal(t, expectedCode, res.StatusCode)
	if expectedCode == http.StatusOK {
		var body map[string]string
		err := json.ReadFrom(res.Body, &body)
		require.NoError(t, err)
		token := body[uri.AccessTokenKey]
		require.NotEmpty(t, token)
		return body
	}
	return nil
}

func GetDefaultAccessToken(t *testing.T, opts ...func(opts *AccessTokenOptions)) string {
	resp := GetAccessToken(t, http.StatusOK, opts...)
	return resp[uri.AccessTokenKey]
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
	return jwt.NewValidator(jwt.NewKeyCache(jwkURL, &client))
}

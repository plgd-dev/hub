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

	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/service"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
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

var ServiceOAuthClient = oauthsigner.Client{
	ID:                  "serviceClient",
	SecretFile:          "data:,serviceClientSecret",
	Owner:               "1",
	AccessTokenLifetime: 0,
	AllowedGrantTypes:   []oauthsigner.GrantType{oauthsigner.GrantTypeClientCredentials},
	AllowedAudiences:    nil,
	AllowedScopes:       nil,
	InsertTokenClaims:   map[string]interface{}{"hardcodedClaim": true},
}

var JWTPrivateKeyOAuthClient = oauthsigner.Client{
	ID:                  "JWTPrivateKeyClient",
	SecretFile:          "data:,JWTPrivateKeyClientSecret",
	AccessTokenLifetime: 0,
	AllowedGrantTypes:   []oauthsigner.GrantType{oauthsigner.GrantTypeClientCredentials},
	AllowedAudiences:    nil,
	AllowedScopes:       nil,
	JWTPrivateKey: oauthsigner.PrivateKeyJWTConfig{
		Enabled:       true,
		Authorization: config.MakeValidatorConfig(),
	},
}

var OAuthClients = oauthsigner.OAuthClientsConfig{
	&ServiceOAuthClient,
	&JWTPrivateKeyOAuthClient,
}

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config

	cfg.Log = log.MakeDefaultConfig()

	cfg.APIs.HTTP.Addr = config.M2M_OAUTH_SERVER_HTTP_HOST
	cfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	cfg.Clients.OpenTelemetryCollector = kitNetHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}
	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.M2M_OAUTH_SERVER_HOST)
	cfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	cfg.APIs.GRPC.Authorization.Endpoints = append(cfg.APIs.GRPC.Authorization.Endpoints,
		validator.AuthorityConfig{
			Authority: testHttp.HTTPS_SCHEME + config.M2M_OAUTH_SERVER_HTTP_HOST + uri.Base,
			HTTP:      config.MakeHttpClientConfig(),
		},
	)
	cfg.APIs.GRPC.Authorization.Config = config.MakeValidatorConfig()
	cfg.Clients.Storage = MakeStoreConfig()

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

type AccessTokenOptions struct {
	Host         string
	ClientID     string
	ClientSecret string
	GrantType    string
	Audience     string
	JWT          string
	PostForm     bool
	Expiration   time.Time
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

func WithExpiration(expiration time.Time) func(opts *AccessTokenOptions) {
	return func(opts *AccessTokenOptions) {
		opts.Expiration = expiration
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
		GrantType:    string(oauthsigner.GrantTypeClientCredentials),
		Ctx:          context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}
	reqBody := map[string]interface{}{
		uri.GrantTypeKey: options.GrantType,
		uri.ClientIDKey:  options.ClientID,
	}
	if options.Audience != "" {
		reqBody[uri.AudienceKey] = options.Audience
	}
	if options.JWT != "" {
		reqBody[uri.ClientAssertionKey] = options.JWT
		reqBody[uri.ClientAssertionTypeKey] = uri.ClientAssertionTypeJWT
	}
	if !options.Expiration.IsZero() {
		reqBody[uri.ExpirationKey] = options.Expiration.Unix()
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
	return jwt.NewValidator(jwt.NewKeyCache(jwkURL, &client), log.Get())
}

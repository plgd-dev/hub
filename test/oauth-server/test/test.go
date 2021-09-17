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
	"testing"

	"github.com/plgd-dev/kit/codec/json"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.HTTP = config.MakeListenerConfig(config.OAUTH_SERVER_HOST)
	cfg.APIs.HTTP.TLS.ClientCertificateRequired = false

	cfg.OAuthSigner.IDTokenKeyFile = os.Getenv("TEST_OAUTH_SERVER_ID_TOKEN_PRIVATE_KEY")
	cfg.OAuthSigner.AccessTokenKeyFile = os.Getenv("TEST_OAUTH_SERVER_ACCESS_TOKEN_PRIVATE_KEY")
	cfg.OAuthSigner.Domain = config.OAUTH_SERVER_HOST

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg)

	return cfg
}

func SetUp(t *testing.T) (TearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config) func() {
	ctx := context.Background()
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()
	return func() {
		_ = s.Shutdown()
		wg.Wait()
	}
}

func NewRequest(method, url string, body io.Reader) *requestBuilder {
	b := requestBuilder{
		method:      method,
		body:        body,
		uri:         fmt.Sprintf("https://%s%s", config.OAUTH_SERVER_HOST, url),
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string]string),
	}
	return &b
}

type requestBuilder struct {
	method      string
	body        io.Reader
	uri         string
	uriParams   map[string]interface{}
	header      map[string]string
	queryParams map[string]string
}

func (c *requestBuilder) AddQuery(key, value string) *requestBuilder {
	c.queryParams[key] = value
	return c
}

func (c *requestBuilder) Build() *http.Request {
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

func HTTPDo(t *testing.T, req *http.Request, followRedirect bool) *http.Response {
	trans := http.DefaultTransport.(*http.Transport).Clone()
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	c := &http.Client{
		Transport: trans,
	}
	if !followRedirect {
		c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

func GetServiceTokenForClient(t *testing.T, clientId string) string {
	reqBody := map[string]string{
		"grant_type":    string(service.AllowedGrantType_CLIENT_CREDENTIALS),
		uri.ClientIDKey: clientId,
		"audience":      "localhost",
	}
	d, err := json.Encode(reqBody)
	require.NoError(t, err)

	getReq := NewRequest(http.MethodPost, uri.Token, bytes.NewReader(d)).Build()
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

func GetServiceToken(t *testing.T) string {
	return GetServiceTokenForClient(t, service.ClientTest)
}

func GetDeviceAuthorizationCodeForClient(t *testing.T, clientId, deviceID string) string {
	u, err := url.Parse(uri.Authorize)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	q.Add(uri.ClientIDKey, clientId)
	if deviceID != "" {
		q.Add(uri.DeviceId, deviceID)
	}
	u.RawQuery = q.Encode()
	getReq := NewRequest(http.MethodGet, u.String(), nil).Build()
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

func GetDeviceAuthorizationCode(t *testing.T, deviceID string) string {
	return GetDeviceAuthorizationCodeForClient(t, service.ClientTest, deviceID)
}

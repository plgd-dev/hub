package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	authURI "github.com/go-ocf/cloud/authorization/uri"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/refImpl"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/test/service"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/net/http/transport"
	"github.com/jtacoma/uritemplates"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

const Cloud2cloud_GW_HOST = "localhost:9090"

func SetUp(ctx context.Context, t *testing.T) (TearDown func()) {
	tearDown := grpcTest.SetUp(ctx, t)

	var cfg refImpl.Config
	err := envconfig.Process("", &cfg)
	require.NoError(t, err)
	cfg.Service.Addr = Cloud2cloud_GW_HOST
	cfg.JwksURL = "https://" + grpcTest.AUTH_HTTP_HOST + authURI.JWKs
	cfg.Service.AuthServerAddr = grpcTest.AUTH_HOST
	cfg.Service.ResourceAggregateAddr = grpcTest.RESOURCE_AGGREGATE_HOST
	cfg.Service.ResourceDirectoryAddr = grpcTest.RESOURCE_DIRECTORY_HOST
	cfg.Service.FQDN = "cloud2cloud-gateway-" + t.Name()
	cfg.Listen.Acme.DisableVerifyClientCertificate = true
	c2cShutdown := service.NewCloud2cloudGateway(t, cfg)

	return func() {
		c2cShutdown()
		tearDown()
	}
}

func NewRequest(method, url string, body io.Reader) *requestBuilder {
	b := requestBuilder{
		method:      method,
		body:        body,
		uri:         fmt.Sprintf("https://%s%s", Cloud2cloud_GW_HOST, url),
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

func (c *requestBuilder) AuthToken(token string) *requestBuilder {
	c.header["Authorization"] = fmt.Sprintf("bearer %s", token)
	return c
}

func (c *requestBuilder) AddQuery(key, value string) *requestBuilder {
	c.queryParams[key] = value
	return c
}

func (c *requestBuilder) AddHeader(key, value string) *requestBuilder {
	c.header[key] = value
	return c
}

func (c *requestBuilder) Build(ctx context.Context, t *testing.T) *http.Request {
	tmp, err := uritemplates.Parse(c.uri)
	require.NoError(t, err)
	uri, err := tmp.Expand(c.uriParams)
	require.NoError(t, err)
	url, err := url.Parse(uri)
	require.NoError(t, err)
	query := url.Query()

	token, err := kitNetGrpc.TokenFromOutgoingMD(ctx)
	if err == nil {
		c.AuthToken(token)
	}

	for k, v := range c.queryParams {
		query.Set(k, v)
	}
	url.RawQuery = query.Encode()
	request, _ := http.NewRequestWithContext(ctx, c.method, url.String(), c.body)
	for k, v := range c.header {
		request.Header.Add(k, v)
	}
	return request
}

func DoHTTPRequest(t *testing.T, req *http.Request) *http.Response {
	trans := transport.NewDefaultTransport()
	trans.TLSClientConfig = &tls.Config{
		RootCAs: grpcTest.GetRootCertificatePool(t),
	}
	c := http.Client{
		Transport: trans,
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

const TestTimeout = 20 * time.Second

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.HTTP_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false

	cfg.Clients.GrpcGateway.Connection = config.MakeGrpcClientConfig(config.GRPC_HOST)

	err := cfg.Validate()
	require.NoError(t, err)

	fmt.Printf("cfg\n%v\n", cfg.String())

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
		s.Serve()
	}()
	return func() {
		s.Shutdown()
		wg.Wait()
	}
}

func NewRequest(method, url string, body io.Reader) *requestBuilder {
	b := requestBuilder{
		method:      method,
		body:        body,
		uri:         fmt.Sprintf("https://%v%v", config.HTTP_GW_HOST, url),
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string][]string),
	}
	return &b
}

type requestBuilder struct {
	method       string
	body         io.Reader
	uri          string
	uriParams    map[string]interface{}
	header       map[string]string
	queryParams  map[string][]string
	resourceHref string
	query        string
}

func (c *requestBuilder) DeviceId(deviceID string) *requestBuilder {
	c.uriParams[uri.DeviceIDKey] = deviceID
	return c
}

func (c *requestBuilder) Shadow(v bool) *requestBuilder {
	c.AddQuery(uri.ShadowQueryKey, fmt.Sprintf("%v", v))
	return c
}

func (c *requestBuilder) ResourceInterface(v string) *requestBuilder {
	if v == "" {
		return c
	}
	c.AddQuery(uri.InterfaceQueryKey, v)
	return c
}

func (c *requestBuilder) ResourceHref(resourceHref string) *requestBuilder {
	if len(resourceHref) > 0 && resourceHref[0] == '/' {
		resourceHref = resourceHref[1:]
	}
	c.resourceHref = resourceHref
	return c
}

func (c *requestBuilder) AuthToken(token string) *requestBuilder {
	c.header["Authorization"] = fmt.Sprintf("bearer %s", token)
	return c
}

func (c *requestBuilder) AcceptContent(acceptContent string) *requestBuilder {
	if acceptContent == "" {
		return c
	}
	c.header["Accept-Content"] = acceptContent
	return c
}

func (c *requestBuilder) AddQuery(key string, value ...string) *requestBuilder {
	c.queryParams[key] = append(c.queryParams[key], value...)
	return c
}

func (c *requestBuilder) AddTypeFilter(typeFilter []string) *requestBuilder {
	if len(typeFilter) == 0 {
		return c
	}
	c.AddQuery(uri.TypeFilterQueryKey, typeFilter...)
	return c
}

func (c *requestBuilder) AddCommandsFilter(commandsFilter []string) *requestBuilder {
	if len(commandsFilter) == 0 {
		return c
	}
	c.AddQuery(uri.CommandsFilterQueryKey, commandsFilter...)
	return c
}

func (c *requestBuilder) SetQuery(value string) *requestBuilder {
	c.query = value
	return c
}

func (c *requestBuilder) Build() *http.Request {
	uri := strings.Replace(c.uri, "{"+uri.ResourceHrefKey+"}", c.resourceHref, -1)

	tmp, _ := uritemplates.Parse(uri)
	uri, _ = tmp.Expand(c.uriParams)
	url, _ := url.Parse(uri)
	query := url.Query()
	for k, vals := range c.queryParams {
		for _, v := range vals {
			query.Add(k, v)
		}
	}
	if c.query != "" {
		url.RawQuery = c.query
	} else {
		url.RawQuery = query.Encode()
	}
	fmt.Printf("URL %v\n", url.String())
	request, _ := http.NewRequest(c.method, url.String(), c.body)
	for k, v := range c.header {
		request.Header.Add(k, v)
	}
	return request
}

func HTTPDo(t *testing.T, req *http.Request) *http.Response {
	trans := http.DefaultTransport.(*http.Transport).Clone()
	trans.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	c := http.Client{
		Transport: trans,
	}
	resp, err := c.Do(req)
	require.NoError(t, err)
	return resp
}

type SortLinksByHref []interface{}

func (a SortLinksByHref) Len() int      { return len(a) }
func (a SortLinksByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortLinksByHref) Less(i, j int) bool {
	e1 := a[i].(map[interface{}]interface{})
	e2 := a[j].(map[interface{}]interface{})
	return e1["href"].(string) < e2["href"].(string)
}

func SortLinks(s []interface{}) []interface{} {
	v := SortLinksByHref(s)
	sort.Sort(v)
	return v
}

func CleanUpDeviceRepresentation(v interface{}) interface{} {
	d, ok := v.(map[interface{}]interface{})
	if !ok {
		return v
	}
	device, ok := d["device"].(map[interface{}]interface{})
	if ok {
		delete(device, "piid")
	}
	metadata, ok := d["metadata"].(map[interface{}]interface{})
	if ok {
		connectionStatus, ok := metadata["connectionStatus"].(map[interface{}]interface{})
		if ok {
			delete(connectionStatus, "validUntil")
		}
	}
	links, ok := d["links"].([]interface{})
	if !ok {
		return v
	}
	links = SortLinks(links)
	for _, l := range links {
		li, ok := l.(map[interface{}]interface{})
		if !ok {
			continue
		}
		delete(li, "ins")
		delete(li, "id")
	}
	d["links"] = links
	return d
}

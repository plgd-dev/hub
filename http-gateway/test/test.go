package test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/go-ocf/kit/net/http/transport"

	authURI "github.com/go-ocf/cloud/authorization/uri"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/go-ocf/cloud/http-gateway/service"
	"github.com/go-ocf/cloud/http-gateway/uri"
	"github.com/jtacoma/uritemplates"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

const HTTP_GW_Port = 7000
const HTTP_GW_Host = "0.0.0.0"
const TestTimeout = 10 * time.Second

func NewTestBackendConfig() service.Config {
	var cfg service.Config
	envconfig.Process("", &cfg)
	cfg.AccessTokenURL = grpcTest.AUTH_HOST
	cfg.Address = fmt.Sprintf("%s:%d", HTTP_GW_Host, HTTP_GW_Port)
	cfg.Listen.Acme.DisableVerifyClientCertificate = true
	cfg.DefaultRequestTimeout = time.Second * 3
	cfg.JwksURL = "https://" + grpcTest.AUTH_HTTP_HOST + authURI.JWKs
	cfg.Service.AuthServerAddr = grpcTest.AUTH_HOST
	cfg.Service.ResourceAggregateAddr = grpcTest.RESOURCE_AGGREGATE_HOST
	cfg.Service.ResourceDirectoryAddr = grpcTest.RESOURCE_DIRECTORY_HOST
	cfg.Service.FQDN = "http-gateway"
	cfg.UserDevicesManagerExpiration = time.Second * 1
	cfg.UserDevicesManagerTickFrequency = time.Millisecond * 500
	cfg.Service.OAuth.Endpoint.TokenURL = "https://" + grpcTest.AUTH_HTTP_HOST + "/api/authz/token"
	return cfg
}

func NewTestHTTPGW(t *testing.T, config string) func() {
	service, err := service.New(config)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		service.Serve()
	}()
	return func() {
		service.Shutdown()
		wg.Wait()
	}
}

func GetTestRebootUri(deviceID string, t *testing.T) string {
	template, _ := uritemplates.Parse(fmt.Sprintf("https://localhost:%d%s", HTTP_GW_Port, uri.DeviceReboot))
	values := make(map[string]interface{})
	values[uri.DeviceIDKey] = deviceID
	u, _ := template.Expand(values)
	t.Log("Reboot request URI: ", u)
	return u
}

func NewRequest(method, url string, body io.Reader) *requestBuilder {
	b := requestBuilder{
		method:      method,
		body:        body,
		uri:         fmt.Sprintf("https://localhost:%d%s", HTTP_GW_Port, url),
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

func (c *requestBuilder) DeviceId(deviceID string) *requestBuilder {
	c.uriParams[uri.DeviceIDKey] = deviceID
	return c
}

func (c *requestBuilder) AuthToken(token string) *requestBuilder {
	c.header["Authorization"] = fmt.Sprintf("bearer %s", token)
	return c
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

func HTTPDo(t *testing.T, req *http.Request) *http.Response {
	trans := transport.NewDefaultTransport()
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
	}
	d["links"] = links
	return d
}

func GetDeviceRepresentation(deviceID string, deviceName string) interface{} {
	return CleanUpDeviceRepresentation(map[interface{}]interface{}{"device": map[interface{}]interface{}{"di": "" + deviceID + "", "dmn": []interface{}{}, "dmno": "", "if": interface{}(nil), "n": "" + deviceName + "", "rt": interface{}(nil)}, "links": []interface{}{map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/light/1", "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "rt": []interface{}{"core.light"}}, map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/light/2", "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "rt": []interface{}{"core.light"}}, map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/oc/con", "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "rt": []interface{}{"oic.wk.con"}}, map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/oic/cloud/s", "if": []interface{}{"oic.if.baseline"}, "rt": []interface{}{"x.cloud.device.status"}}, map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/oic/d", "if": []interface{}{"oic.if.r", "oic.if.baseline"}, "rt": []interface{}{"oic.d.cloudDevice", "oic.wk.d"}}, map[interface{}]interface{}{"di": "" + deviceID + "", "href": "/oic/p", "if": []interface{}{"oic.if.r", "oic.if.baseline"}, "rt": []interface{}{"oic.wk.p"}}}, "status": "online"})
}

package test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/cloud/http-gateway/service"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

const TestTimeout = 20 * time.Second

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.APIs.HTTP.Authorization = config.MakeAuthorizationConfig()
	cfg.APIs.HTTP.WebSocket.ReadLimit = 8 * 1024
	cfg.APIs.HTTP.WebSocket.ReadTimeout = time.Second * 4
	cfg.APIs.HTTP.Connection = config.MakeListenerConfig(config.HTTP_GW_HOST)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false

	cfg.Clients.ResourceAggregate.Connection = config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST)
	cfg.Clients.ResourceDirectory.Connection = config.MakeGrpcClientConfig(config.RESOURCE_DIRECTORY_HOST)

	cfg.Clients.Eventbus.NATS = config.MakeSubscriberConfig()
	cfg.Clients.Eventbus.GoPoolSize = 16

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

func GetTestRebootUri(deviceID string, t *testing.T) string {
	template, _ := uritemplates.Parse(fmt.Sprintf("https://%v%v", config.HTTP_GW_HOST, uri.DeviceReboot))
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
		uri:         fmt.Sprintf("https://%v%v", config.HTTP_GW_HOST, url),
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

func GetDeviceRepresentation(deviceID string, deviceName string) interface{} {
	return CleanUpDeviceRepresentation(
		map[interface{}]interface{}{
			"device": map[interface{}]interface{}{
				"di":   deviceID,
				"dmn":  []interface{}{},
				"dmno": "",
				"if":   []interface{}{"oic.if.r", "oic.if.baseline"},
				"n":    deviceName,
				"rt":   []interface{}{"oic.d.cloudDevice", "oic.wk.d"}},
			"links": []interface{}{map[interface{}]interface{}{"di": deviceID, "href": "/light/1", "id": commands.MakeStatusResourceUUID(deviceID), "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "p": map[interface{}]interface{}{"bm": uint64(3), "port": uint64(0), "sec": false, "x.org.iotivity.tcp": uint64(0), "x.org.iotivity.tls": uint64(0)}, "rt": []interface{}{"core.light"}}, map[interface{}]interface{}{"di": deviceID, "href": "/light/2", "id": "d72a96bd-449a-51f1-a7d3-71f4ccad8d00", "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "p": map[interface{}]interface{}{"bm": uint64(3), "port": uint64(0), "sec": false, "x.org.iotivity.tcp": uint64(0), "x.org.iotivity.tls": uint64(0)}, "rt": []interface{}{"core.light"}}, map[interface{}]interface{}{"di": deviceID, "href": "/oc/con", "id": "b67202b3-6bd7-5f0b-ada0-83b7e985cef4", "if": []interface{}{"oic.if.rw", "oic.if.baseline"}, "p": map[interface{}]interface{}{"bm": uint64(3), "port": uint64(0), "sec": false, "x.org.iotivity.tcp": uint64(0), "x.org.iotivity.tls": uint64(0)}, "rt": []interface{}{"oic.wk.con"}}, map[interface{}]interface{}{"di": deviceID, "href": "/oic/d", "id": "7013279c-4f28-503f-9425-d76ae580c590", "if": []interface{}{"oic.if.r", "oic.if.baseline"}, "p": map[interface{}]interface{}{"bm": uint64(3), "port": uint64(0), "sec": false, "x.org.iotivity.tcp": uint64(0), "x.org.iotivity.tls": uint64(0)}, "rt": []interface{}{"oic.d.cloudDevice", "oic.wk.d"}}, map[interface{}]interface{}{"di": deviceID, "href": "/oic/p", "id": "d6940aea-d1b5-53dd-a123-838b73b02d10", "if": []interface{}{"oic.if.r", "oic.if.baseline"}, "p": map[interface{}]interface{}{"bm": uint64(3), "port": uint64(0), "sec": false, "x.org.iotivity.tcp": uint64(0), "x.org.iotivity.tls": uint64(0)}, "rt": []interface{}{"oic.wk.p"}}},
			"metadata": map[interface{}]interface{}{
				"connectionStatus": map[interface{}]interface{}{
					"value": "online",
				},
				"shadowSynchronization": map[interface{}]interface{}{
					"disabled": false,
				},
			}},
	)
}

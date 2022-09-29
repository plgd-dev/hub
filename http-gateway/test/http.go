package test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func NewRequest(method, path string, body io.Reader) *requestBuilder {
	b := requestBuilder{
		method:      method,
		body:        body,
		host:        config.HTTP_GW_HOST,
		path:        path,
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string][]string),
	}
	return &b
}

type requestBuilder struct {
	method       string
	body         io.Reader
	host         string
	path         string
	uriParams    map[string]interface{}
	header       map[string]string
	queryParams  map[string][]string
	resourceHref string
	query        string
}

func (c *requestBuilder) Host(host string) *requestBuilder {
	c.host = host
	return c
}

func (c *requestBuilder) DeviceId(deviceID string) *requestBuilder {
	c.uriParams[uri.DeviceIDKey] = deviceID
	return c
}

func (c *requestBuilder) Shadow(v bool) *requestBuilder {
	c.AddQuery(uri.ShadowQueryKey, fmt.Sprintf("%v", v))
	return c
}

func (c *requestBuilder) Timestamp(v time.Time) *requestBuilder {
	if v.IsZero() {
		return c
	}
	c.AddQuery(uri.TimestampFilterQueryKey, fmt.Sprintf("%v", v.UnixNano()))
	return c
}

func (c *requestBuilder) ResourceInterface(v string) *requestBuilder {
	if v == "" {
		return c
	}
	c.AddQuery(uri.ResourceInterfaceQueryKey, v)
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

func (c *requestBuilder) Accept(accept string) *requestBuilder {
	if accept == "" {
		return c
	}
	c.header["Accept"] = accept
	return c
}

func (c *requestBuilder) ContentType(contentType string) *requestBuilder {
	if contentType == "" {
		return c
	}
	c.header[uri.ContentTypeHeaderKey] = contentType
	return c
}

func (c *requestBuilder) AddQuery(key string, value ...string) *requestBuilder {
	c.queryParams[key] = append(c.queryParams[key], value...)
	return c
}

func (c *requestBuilder) AddDeviceIdFilter(deviceFilter []string) *requestBuilder {
	if len(deviceFilter) == 0 {
		return c
	}
	c.AddQuery(uri.DeviceIdFilterQueryKey, deviceFilter...)
	return c
}

func (c *requestBuilder) AddResourceIdFilter(resourcefilter []string) *requestBuilder {
	if len(resourcefilter) == 0 {
		return c
	}
	c.AddQuery(uri.ResourceIdFilterQueryKey, resourcefilter...)
	return c
}

func (c *requestBuilder) AddStatusFilter(statusFilter []string) *requestBuilder {
	if len(statusFilter) == 0 {
		return c
	}
	c.AddQuery(uri.StatusFilterQueryKey, statusFilter...)
	return c
}

func (c *requestBuilder) AddTypeFilter(typeFilter []string) *requestBuilder {
	if len(typeFilter) == 0 {
		return c
	}
	c.AddQuery(uri.TypeFilterQueryKey, typeFilter...)
	return c
}

func (c *requestBuilder) AddCorrelationIdFilter(correlationID []string) *requestBuilder {
	if len(correlationID) == 0 {
		return c
	}
	c.AddQuery(uri.CorrelationIdFilterQueryKey, correlationID...)
	return c
}

func (c *requestBuilder) AddCommandsFilter(commandFilter []string) *requestBuilder {
	if len(commandFilter) == 0 {
		return c
	}
	c.AddQuery(uri.CommandFilterQueryKey, commandFilter...)
	return c
}

func (c *requestBuilder) AddTimeToLive(ttl time.Duration) *requestBuilder {
	if ttl == 0 {
		return c
	}
	c.AddQuery(uri.TimeToLiveQueryKey, strconv.FormatInt(ttl.Nanoseconds(), 10))
	return c
}

func (c *requestBuilder) SetQuery(value string) *requestBuilder {
	c.query = value
	return c
}

func (c *requestBuilder) Build() *http.Request {
	r := fmt.Sprintf("https://%v%v", c.host, c.path)
	uri := strings.ReplaceAll(r, "{"+uri.ResourceHrefKey+"}", c.resourceHref)

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

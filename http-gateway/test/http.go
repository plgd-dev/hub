package test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func NewRequest(method, path string, body io.Reader) *RequestBuilder {
	b := RequestBuilder{
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

type RequestBuilder struct {
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

func (c *RequestBuilder) Host(host string) *RequestBuilder {
	c.host = host
	return c
}

func (c *RequestBuilder) DeviceId(deviceID string) *RequestBuilder {
	c.uriParams[uri.DeviceIDKey] = deviceID
	return c
}

func (c *RequestBuilder) Twin(v bool) *RequestBuilder {
	c.AddQuery(uri.TwinQueryKey, strconv.FormatBool(v))
	return c
}

func (c *RequestBuilder) Timestamp(v time.Time) *RequestBuilder {
	if v.IsZero() {
		return c
	}
	c.AddQuery(uri.TimestampFilterQueryKey, strconv.FormatInt(v.UnixNano(), 10))
	return c
}

func (c *RequestBuilder) ResourceInterface(v string) *RequestBuilder {
	if v == "" {
		return c
	}
	c.AddQuery(uri.ResourceInterfaceQueryKey, v)
	return c
}

func (c *RequestBuilder) ETag(v []byte) *RequestBuilder {
	if len(v) == 0 {
		return c
	}
	c.header[pkgHttp.ETagHeaderKey] = base64.StdEncoding.EncodeToString(v)
	return c
}

func (c *RequestBuilder) ETags(v [][]byte) *RequestBuilder {
	if len(v) == 0 {
		return c
	}
	for _, e := range v {
		c.AddQuery(uri.ETagQueryKey, base64.StdEncoding.EncodeToString(e))
	}
	return c
}

func (c *RequestBuilder) ResourceHref(resourceHref string) *RequestBuilder {
	if len(resourceHref) > 0 && resourceHref[0] == '/' {
		resourceHref = resourceHref[1:]
	}
	c.resourceHref = resourceHref
	return c
}

func (c *RequestBuilder) AuthToken(token string) *RequestBuilder {
	c.header["Authorization"] = "bearer " + token
	return c
}

func (c *RequestBuilder) Accept(accept string) *RequestBuilder {
	if accept == "" {
		return c
	}
	c.header["Accept"] = accept
	return c
}

func (c *RequestBuilder) OnlyContent(v bool) *RequestBuilder {
	c.AddQuery(uri.OnlyContentQueryKey, strconv.FormatBool(v))
	return c
}

func (c *RequestBuilder) ContentType(contentType string) *RequestBuilder {
	if contentType == "" {
		return c
	}
	c.header[pkgHttp.ContentTypeHeaderKey] = contentType
	return c
}

func (c *RequestBuilder) AddQuery(key string, value ...string) *RequestBuilder {
	c.queryParams[key] = append(c.queryParams[key], value...)
	return c
}

func (c *RequestBuilder) AddDeviceIdFilter(deviceFilter []string) *RequestBuilder {
	if len(deviceFilter) == 0 {
		return c
	}
	c.AddQuery(uri.DeviceIdFilterQueryKey, deviceFilter...)
	return c
}

func (c *RequestBuilder) AddResourceIdFilter(resourceFilter []*pb.ResourceIdFilter) *RequestBuilder {
	if len(resourceFilter) == 0 {
		return c
	}
	resourceFilterStr := make([]string, 0, len(resourceFilter))
	for _, f := range resourceFilter {
		resourceFilterStr = append(resourceFilterStr, f.ToString())
	}
	c.AddQuery(uri.ResourceIdFilterQueryKey, resourceFilterStr...)
	return c
}

func (c *RequestBuilder) AddStatusFilter(statusFilter []string) *RequestBuilder {
	if len(statusFilter) == 0 {
		return c
	}
	c.AddQuery(uri.StatusFilterQueryKey, statusFilter...)
	return c
}

func (c *RequestBuilder) AddTypeFilter(typeFilter []string) *RequestBuilder {
	if len(typeFilter) == 0 {
		return c
	}
	c.AddQuery(uri.TypeFilterQueryKey, typeFilter...)
	return c
}

func (c *RequestBuilder) AddCorrelationIdFilter(correlationID []string) *RequestBuilder {
	if len(correlationID) == 0 {
		return c
	}
	c.AddQuery(uri.CorrelationIdFilterQueryKey, correlationID...)
	return c
}

func (c *RequestBuilder) AddCommandsFilter(commandFilter []string) *RequestBuilder {
	if len(commandFilter) == 0 {
		return c
	}
	c.AddQuery(uri.CommandFilterQueryKey, commandFilter...)
	return c
}

func (c *RequestBuilder) AddTimeToLive(ttl time.Duration) *RequestBuilder {
	if ttl == 0 {
		return c
	}
	c.AddQuery(uri.TimeToLiveQueryKey, strconv.FormatInt(ttl.Nanoseconds(), 10))
	return c
}

func (c *RequestBuilder) AddIssuerID(issuerID string) *RequestBuilder {
	c.uriParams[uri.IssuerIDKey] = issuerID
	return c
}

func (c *RequestBuilder) SetQuery(value string) *RequestBuilder {
	c.query = value
	return c
}

func (c *RequestBuilder) Build() *http.Request {
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
	request, _ := http.NewRequestWithContext(context.Background(), c.method, url.String(), c.body)
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

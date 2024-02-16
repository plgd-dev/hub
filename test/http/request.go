package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

type HTTPRequestBuilder struct {
	method       string
	body         io.Reader
	uri          string
	uriParams    map[string]interface{}
	header       map[string]string
	queryParams  map[string][]string
	resourceHref string
	query        string
}

func NewHTTPRequest(method, url string, body io.Reader) *HTTPRequestBuilder {
	b := HTTPRequestBuilder{
		method:      method,
		body:        body,
		uri:         url,
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string][]string),
	}
	return &b
}

func (c *HTTPRequestBuilder) AuthToken(token string) *HTTPRequestBuilder {
	c.header["Authorization"] = fmt.Sprintf("bearer %s", token)
	return c
}

func (c *HTTPRequestBuilder) AddContentQuery(content string) *HTTPRequestBuilder {
	c.AddQuery(ContentQueryKey, content)
	return c
}

func (c *HTTPRequestBuilder) AddQuery(key string, value ...string) *HTTPRequestBuilder {
	c.queryParams[key] = append(c.queryParams[key], value...)
	return c
}

func (c *HTTPRequestBuilder) Accept(accept string) *HTTPRequestBuilder {
	if accept == "" {
		return c
	}
	c.header["Accept"] = accept
	return c
}

func (c *HTTPRequestBuilder) DeviceId(deviceID string) *HTTPRequestBuilder {
	if deviceID == "" {
		return c
	}
	c.uriParams[DeviceIDKey] = deviceID
	return c
}

func (c *HTTPRequestBuilder) ResourceHref(resourceHref string) *HTTPRequestBuilder {
	if resourceHref == "" {
		return c
	}
	if len(resourceHref) > 0 && resourceHref[0] == '/' {
		resourceHref = resourceHref[1:]
	}
	c.resourceHref = resourceHref
	return c
}

func (c *HTTPRequestBuilder) SubscriptionID(subscriptionID string) *HTTPRequestBuilder {
	if subscriptionID == "" {
		return c
	}
	c.uriParams[SubscriptionIDKey] = subscriptionID
	return c
}

func (c *HTTPRequestBuilder) SetQuery(value string) *HTTPRequestBuilder {
	c.query = value
	return c
}

func (c *HTTPRequestBuilder) Build(ctx context.Context, t *testing.T) *http.Request {
	u := c.uri
	if len(c.resourceHref) > 0 {
		u = strings.ReplaceAll(c.uri, "{"+ResourceHrefKey+"}", c.resourceHref)
	}

	tmp, err := uritemplates.Parse(u)
	require.NoError(t, err)
	uri, err := tmp.Expand(c.uriParams)
	require.NoError(t, err)
	url, err := url.Parse(uri)
	require.NoError(t, err)
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
	url.RawQuery = query.Encode()
	fmt.Printf("URL %v\n", url.String())
	request, _ := http.NewRequestWithContext(ctx, c.method, url.String(), c.body)
	for k, v := range c.header {
		request.Header.Add(k, v)
	}
	return request
}

func DoHTTPRequest(t *testing.T, req *http.Request) *http.Response {
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

func ReadHTTPResponse(t *testing.T, w io.Reader, contentType string, data interface{}) {
	readFrom := func(_ io.Reader, _ interface{}) error {
		return fmt.Errorf("not supported")
	}
	switch contentType {
	case message.AppJSON.String():
		readFrom = json.ReadFrom
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		readFrom = cbor.ReadFrom
	case "text/plain":
		readFrom = func(w io.Reader, v interface{}) error {
			b, err := io.ReadAll(w)
			if err != nil {
				return err
			}
			val := reflect.ValueOf(v)
			if val.Kind() != reflect.Ptr {
				return fmt.Errorf("some: check must be a pointer")
			}
			val.Elem().Set(reflect.ValueOf(string(b)))
			return nil
		}
	}
	err := readFrom(w, data)
	require.NoError(t, err)
}

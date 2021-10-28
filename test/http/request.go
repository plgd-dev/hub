package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/go-coap/v2/message"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
)

const HTTPS_SCHEME = "https://"

type HTTPRequestBuilder struct {
	method      string
	body        io.Reader
	uri         string
	uriParams   map[string]interface{}
	header      map[string]string
	queryParams map[string]string
}

func NewHTTPRequest(method, url string, body io.Reader) *HTTPRequestBuilder {
	b := HTTPRequestBuilder{
		method:      method,
		body:        body,
		uri:         url,
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string]string),
	}
	return &b
}

func (c *HTTPRequestBuilder) AuthToken(token string) *HTTPRequestBuilder {
	c.header["Authorization"] = fmt.Sprintf("bearer %s", token)
	return c
}

func (c *HTTPRequestBuilder) AddQuery(key, value string) *HTTPRequestBuilder {
	c.queryParams[key] = value
	return c
}

func (c *HTTPRequestBuilder) Accept(accept string) *HTTPRequestBuilder {
	if accept == "" {
		return c
	}
	c.header["Accept"] = accept
	return c
}

func (c *HTTPRequestBuilder) Build(ctx context.Context, t *testing.T) *http.Request {
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

func ReadHTTPResponse(t *testing.T, w io.Reader, contentType string) interface{} {
	var data interface{}
	readFrom := func(w io.Reader, v interface{}) error {
		return fmt.Errorf("not supported")
	}
	switch contentType {
	case message.AppJSON.String():
		readFrom = json.ReadFrom
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		readFrom = cbor.ReadFrom
	case "text/plain":
		readFrom = func(w io.Reader, v interface{}) error {
			b, err := ioutil.ReadAll(w)
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
	err := readFrom(w, &data)
	require.NoError(t, err)

	return data
}

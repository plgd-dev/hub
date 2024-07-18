package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func GetContentData(content *pb.Content, desiredContentType string) ([]byte, error) {
	if desiredContentType == pkgHttp.ApplicationProtoJsonContentType {
		data, err := protojson.Marshal(content)
		if err != nil {
			return nil, err
		}
		return data, err
	}
	if desiredContentType == message.AppJSON.String() {
		v, err := cbor.ToJSON(content.GetData())
		if err != nil {
			return nil, err
		}
		return []byte(v), err
	}
	return nil, errors.New("not supported")
}

type RequestBuilder struct {
	method       string
	body         io.Reader
	uri          string
	uriParams    map[string]interface{}
	header       map[string]string
	queryParams  map[string][]string
	resourceHref string
	query        string
}

func NewRequest(method, url string, body io.Reader) *RequestBuilder {
	b := RequestBuilder{
		method:      method,
		body:        body,
		uri:         url,
		uriParams:   make(map[string]interface{}),
		header:      make(map[string]string),
		queryParams: make(map[string][]string),
	}
	return &b
}

func (c *RequestBuilder) AuthToken(token string) *RequestBuilder {
	c.header["Authorization"] = "bearer " + token
	return c
}

func (c *RequestBuilder) AddContentQuery(content string) *RequestBuilder {
	c.AddQuery(ContentQueryKey, content)
	return c
}

func (c *RequestBuilder) AddQuery(key string, value ...string) *RequestBuilder {
	c.queryParams[key] = append(c.queryParams[key], value...)
	return c
}

func (c *RequestBuilder) Accept(accept string) *RequestBuilder {
	if accept == "" {
		return c
	}
	c.header[pkgHttp.AcceptHeaderKey] = accept
	return c
}

func (c *RequestBuilder) ContentType(contentType string) *RequestBuilder {
	if contentType == "" {
		return c
	}
	c.header[pkgHttp.ContentTypeHeaderKey] = contentType
	return c
}

func (c *RequestBuilder) DeviceId(deviceID string) *RequestBuilder {
	if deviceID == "" {
		return c
	}
	c.uriParams[DeviceIDKey] = deviceID
	return c
}

func (c *RequestBuilder) ID(id string) *RequestBuilder {
	if id == "" {
		return c
	}
	c.uriParams[IDKey] = id
	return c
}

func (c *RequestBuilder) ResourceHref(resourceHref string) *RequestBuilder {
	if resourceHref == "" {
		return c
	}
	if len(resourceHref) > 0 && resourceHref[0] == '/' {
		resourceHref = resourceHref[1:]
	}
	c.resourceHref = resourceHref
	return c
}

func (c *RequestBuilder) SubscriptionID(subscriptionID string) *RequestBuilder {
	if subscriptionID == "" {
		return c
	}
	c.uriParams[SubscriptionIDKey] = subscriptionID
	return c
}

func (c *RequestBuilder) Version(version string) *RequestBuilder {
	if version == "" {
		return c
	}
	c.AddQuery(VersionKey, version)
	return c
}

func (c *RequestBuilder) IDFilter(idFilter []string) *RequestBuilder {
	if len(idFilter) == 0 {
		return c
	}
	c.AddQuery(IDFilterKey, idFilter...)
	return c
}

func (c *RequestBuilder) SetQuery(value string) *RequestBuilder {
	c.query = value
	return c
}

func (c *RequestBuilder) Build(ctx context.Context, t *testing.T) *http.Request {
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
	request, err := http.NewRequestWithContext(ctx, c.method, url.String(), c.body)
	require.NoError(t, err)
	for k, v := range c.header {
		request.Header.Add(k, v)
	}
	return request
}

func Do(t *testing.T, req *http.Request) *http.Response {
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

func ReadResponse(t *testing.T, w io.Reader, contentType string, data interface{}) {
	readFrom := func(_ io.Reader, _ interface{}) error {
		return errors.New("not supported")
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
				return errors.New("some: check must be a pointer")
			}
			val.Elem().Set(reflect.ValueOf(string(b)))
			return nil
		}
	}
	err := readFrom(w, data)
	require.NoError(t, err)
}

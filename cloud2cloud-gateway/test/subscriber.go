package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type C2CSubscriber struct {
	port      string
	eventsURI string
	subType   SubscriptionType
}

type SubscriptionType int

const (
	SubscriptionType_Devices SubscriptionType = iota
	SubscriptionType_Device
	SubscriptionType_Resource
)

func NewC2CSubscriber(port, eventsURI string, subType SubscriptionType) *C2CSubscriber {
	return &C2CSubscriber{
		port:      port,
		eventsURI: eventsURI,
		subType:   subType,
	}
}

func (c *C2CSubscriber) subscriptionURI() string {
	switch c.subType {
	case SubscriptionType_Devices:
		return C2CURI(uri.DevicesSubscriptions)
	case SubscriptionType_Device:
		return C2CURI(uri.DeviceSubscriptions)
	case SubscriptionType_Resource:
		return C2CURI(uri.ResourceSubscriptions)
	}
	return ""
}

func (c *C2CSubscriber) Subscribe(ctx context.Context, t *testing.T, token, deviceID, href string, eventTypes events.EventTypes) string {
	sub := events.SubscriptionRequest{
		EventsURL:     "https://localhost:" + c.port + c.eventsURI,
		EventTypes:    eventTypes,
		SigningSecret: "a",
	}
	reqData, err := json.Encode(sub)
	require.NoError(t, err)

	uri := c.subscriptionURI()
	require.NotEmpty(t, uri)
	accept := message.AppJSON.String()
	rb := testHttp.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(reqData)).AuthToken(token).Accept(accept)
	rb.DeviceId(deviceID).ResourceHref(href)
	req := rb.Build(ctx, t)
	fmt.Printf("%v\n", req.URL.String())
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	respData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	fmt.Printf("body %v\n", string(respData))
	require.Equal(t, message.AppJSON.String(), resp.Header.Get("Content-Type"))
	require.NotEmpty(t, respData)

	var v map[string]interface{}
	err = json.Decode(respData, &v)
	require.NoError(t, err)
	value, ok := v["subscriptionId"]
	require.True(t, ok)

	subID, ok := value.(string)
	require.True(t, ok)
	return subID
}

func (c *C2CSubscriber) unsubscriptionURI() string {
	switch c.subType {
	case SubscriptionType_Devices:
		return C2CURI(uri.DevicesSubscription)
	case SubscriptionType_Device:
		return C2CURI(uri.DeviceSubscription)
	case SubscriptionType_Resource:
		return C2CURI(uri.ResourceSubscription)
	}
	return ""
}

func (c *C2CSubscriber) Unsubscribe(ctx context.Context, t *testing.T, token, deviceID, href, subID string) {
	uri := c.unsubscriptionURI()
	require.NotEmpty(t, uri)
	rb := testHttp.NewHTTPRequest(http.MethodDelete, uri, nil).AuthToken(token)
	rb.DeviceId(deviceID).ResourceHref(href).SubscriptionID(subID)
	req := rb.Build(ctx, t)
	fmt.Printf("%v\n", req.URL.String())
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	_, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
}

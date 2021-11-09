package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/uri"
	testHttp "github.com/plgd-dev/hub/test/http"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type C2CSubscriber struct {
	port      string
	eventsURI string
}

func NewC2CSubscriber(port, eventsURI string) *C2CSubscriber {
	return &C2CSubscriber{
		port:      port,
		eventsURI: eventsURI,
	}
}

func (c *C2CSubscriber) Subscribe(t *testing.T, ctx context.Context, token, deviceID string, eventTypes events.EventTypes) string {
	sub := events.SubscriptionRequest{
		URL:           "https://localhost:" + c.port + c.eventsURI,
		EventTypes:    eventTypes,
		SigningSecret: "a",
	}
	reqData, err := json.Encode(sub)
	require.NoError(t, err)

	uri := C2CURI(uri.DeviceSubscriptions)
	accept := message.AppJSON.String()
	rb := testHttp.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(reqData)).AuthToken(token).Accept(accept).DeviceId(deviceID)
	req := rb.Build(ctx, t)
	fmt.Printf("%v\n", req.URL.String())
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	respData, err := ioutil.ReadAll(resp.Body)
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

func (c *C2CSubscriber) Unsubscribe(t *testing.T, ctx context.Context, token, deviceID, subID string) {
	uri := C2CURI(uri.DeviceSubscription)
	rb := testHttp.NewHTTPRequest(http.MethodDelete, uri, nil).AuthToken(token).DeviceId(deviceID).SubscriptionID(subID)
	req := rb.Build(ctx, t)
	fmt.Printf("%v\n", req.URL.String())
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	_, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
}

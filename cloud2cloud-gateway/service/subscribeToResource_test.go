package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	testHttp "github.com/plgd-dev/hub/test/http"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_SubscribeToResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	wantCode := http.StatusCreated
	wantContentType := message.AppJSON.String()
	wantContent := true
	wantEventType := events.EventType_ResourceChanged
	wantEventContent := map[interface{}]interface{}{
		"if":    []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		"name":  "Light",
		"power": uint64(0),
		"rt":    []interface{}{types.CORE_LIGHT},
		"state": false,
	}
	eventType := events.EventType_ResourceChanged
	uri := "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1/subscriptions"
	accept := message.AppJSON.String()

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	eventsServer, cleanUpEventsServer := c2cTest.NewTestListener(t)
	defer cleanUpEventsServer()

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc("/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err := events.ParseEventHeader(r)
			assert.NoError(t, err)
			defer func() {
				_ = r.Body.Close()
			}()
			assert.Equal(t, wantEventType, h.EventType)
			buf, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			var v interface{}
			err = json.Decode(buf, &v)
			assert.NoError(t, err)
			assert.Equal(t, wantEventContent, v)
			w.WriteHeader(http.StatusOK)
			err = eventsServer.Close()
			assert.NoError(t, err)
		})).Methods("POST")
		_ = http.Serve(eventsServer, r)
	}()

	_, port, err := net.SplitHostPort(eventsServer.Addr().String())
	require.NoError(t, err)

	sub := events.SubscriptionRequest{
		URL:           "https://localhost:" + port + "/events",
		EventTypes:    events.EventTypes{eventType},
		SigningSecret: "a",
	}
	fmt.Printf("%v\n", uri)

	data, err := json.Encode(sub)
	require.NoError(t, err)
	req := testHttp.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(token).Accept(accept).Build(ctx, t)
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, wantCode, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	v, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("body %v\n", string(v))
	require.NoError(t, err)
	require.Equal(t, wantContentType, resp.Header.Get("Content-Type"))
	if wantContent {
		require.NotEmpty(t, v)
	} else {
		require.Empty(t, v)
	}
}

func TestRequestHandler_SubscribeToResourceTokenTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	services := test.SetUpServicesOAuth | test.SetUpServicesId | test.SetUpServicesCertificateAuthority |
		test.SetUpServicesResourceAggregate | test.SetUpServicesResourceDirectory | test.SetUpServicesGrpcGateway |
		test.SetUpServicesCoapGateway
	tearDown := test.SetUpServices(ctx, t, services)
	defer tearDown()
	c2cgwShutdown := c2cTest.SetUp(t)

	token := oauthTest.GetServiceToken(t, testCfg.OAUTH_SERVER_HOST, oauthTest.ClientTestShortExpiration)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	_, shutdownDevSim := test.OnboardDevSimForClient(ctx, t, c, oauthTest.ClientTestShortExpiration, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	eventsServer, cleanUpEventsServer := c2cTest.NewTestListener(t)
	defer cleanUpEventsServer()

	var cancelled, subscribed atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc("/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err := events.ParseEventHeader(r)
			assert.NoError(t, err)
			defer func() {
				_ = r.Body.Close()
			}()
			_, err = ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			if h.EventType == events.EventType_ResourceChanged {
				w.WriteHeader(http.StatusOK)
				subscribed.Store(true)
				return
			}
			if h.EventType == events.EventType_SubscriptionCanceled {
				w.WriteHeader(http.StatusOK)
				cancelled.Store(true)
				return
			}
			assert.Fail(t, "invalid EventType: %v", h.EventType)
		})).Methods("POST")
		_ = http.Serve(eventsServer, r)
	}()

	uri := "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1/subscriptions"
	_, port, err := net.SplitHostPort(eventsServer.Addr().String())
	require.NoError(t, err)

	sub := events.SubscriptionRequest{
		URL:           "https://localhost:" + port + "/events",
		EventTypes:    events.EventTypes{events.EventType_ResourceChanged},
		SigningSecret: "a",
	}
	fmt.Printf("%v\n", uri)

	data, err := json.Encode(sub)
	require.NoError(t, err)
	accept := message.AppJSON.String()
	req := testHttp.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(token).Accept(accept).Build(ctx, t)
	resp := testHttp.DoHTTPRequest(t, req)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, message.AppJSON.String(), resp.Header.Get("Content-Type"))
	v, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("body %v\n", string(v))
	require.NoError(t, err)

	// let access token expire
	time.Sleep(time.Second * 10)
	assert.True(t, subscribed.Load())
	// stop and start c2c-gw and let it try reestablish resource subscription with expired token
	c2cgwShutdown()
	c2cgwShutdown = c2cTest.SetUp(t)

	// give enough time to wait for cancel subscription response
	time.Sleep(time.Second * 5)
	err = eventsServer.Close()
	assert.NoError(t, err)
	assert.True(t, cancelled.Load())
	defer func() {
		wg.Wait()
		c2cgwShutdown()
	}()
}

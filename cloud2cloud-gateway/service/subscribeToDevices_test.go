package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerSubscribeToDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	wantCode := http.StatusCreated
	wantContentType := message.AppJSON.String()
	wantContent := true
	wantEventType := events.EventType_DevicesOnline
	wantEventContent := []interface{}{
		map[interface{}]interface{}{"di": deviceID},
	}
	eventType := events.EventType_DevicesOnline
	uri := "https://" + config.C2C_GW_HOST + uri.DevicesSubscriptions
	accept := message.AppJSON.String()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	eventsServer, cleanUpEventsServer := c2cTest.NewTestListener(t)
	defer cleanUpEventsServer()

	const eventsURI = "/events"
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc(eventsURI, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err2 := events.ParseEventHeader(r)
			assert.NoError(t, err2)
			defer func() {
				_ = r.Body.Close()
			}()
			assert.Equal(t, wantEventType, h.EventType)
			buf, err2 := io.ReadAll(r.Body)
			assert.NoError(t, err2)
			var v interface{}
			err2 = json.Decode(buf, &v)
			assert.NoError(t, err2)
			assert.Equal(t, wantEventContent, v)
			w.WriteHeader(http.StatusOK)
			err2 = eventsServer.Close()
			assert.NoError(t, err2)
		})).Methods("POST")
		_ = http.Serve(eventsServer, r)
	}()

	_, port, err := net.SplitHostPort(eventsServer.Addr().String())
	require.NoError(t, err)

	sub := events.SubscriptionRequest{
		EventsURL:     "https://localhost:" + port + eventsURI,
		EventTypes:    events.EventTypes{eventType},
		SigningSecret: "a",
	}

	data, err := json.Encode(sub)
	require.NoError(t, err)
	req := testHttp.NewRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(token).Accept(accept).Build(ctx, t)
	resp := testHttp.Do(t, req)
	require.Equal(t, wantCode, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	v, err := io.ReadAll(resp.Body)
	fmt.Printf("body %v\n", string(v))
	require.NoError(t, err)
	require.Equal(t, wantContentType, resp.Header.Get("Content-Type"))
	if wantContent {
		require.NotEmpty(t, v)
	} else {
		require.Empty(t, v)
	}
}

func TestRequestHandlerSubscribeToDevicesOffline(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	wantCode := http.StatusCreated
	wantContentType := message.AppJSON.String()
	wantContent := true
	wantEventType := events.EventType_DevicesOffline
	wantEventContent := []interface{}{}
	eventType := events.EventType_DevicesOffline
	uri := "https://" + config.C2C_GW_HOST + uri.DevicesSubscriptions
	accept := message.AppJSON.String()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	coapgwCfg := coapgwTest.MakeConfig(t)
	coapgwCfg.Log.DumpBody = true
	coapgwCfg.APIs.COAP.Addr = "localhost:45685"
	gwShutdown := coapgwTest.New(t, coapgwCfg)
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+coapgwCfg.APIs.COAP.Addr, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	eventsServer, cleanUpEventsServer := c2cTest.NewTestListener(t)
	defer cleanUpEventsServer()

	const eventsURI = "/events"
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		r := router.NewRouter()
		r.StrictSlash(true)
		r.HandleFunc(eventsURI, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h, err2 := events.ParseEventHeader(r)
			assert.NoError(t, err2)
			defer func() {
				_ = r.Body.Close()
			}()
			assert.Equal(t, wantEventType, h.EventType)
			buf, err2 := io.ReadAll(r.Body)
			assert.NoError(t, err2)
			var v interface{}
			err2 = json.Decode(buf, &v)
			assert.NoError(t, err2)
			assert.Equal(t, wantEventContent, v)
			w.WriteHeader(http.StatusOK)
			err2 = eventsServer.Close()
			assert.NoError(t, err2)
		})).Methods("POST")
		_ = http.Serve(eventsServer, r)
	}()

	_, port, err := net.SplitHostPort(eventsServer.Addr().String())
	require.NoError(t, err)

	sub := events.SubscriptionRequest{
		EventsURL:     "https://localhost:" + port + eventsURI,
		EventTypes:    events.EventTypes{eventType},
		SigningSecret: "a",
	}

	data, err := json.Encode(sub)
	require.NoError(t, err)
	req := testHttp.NewRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(oauthTest.GetDefaultAccessToken(t)).Accept(accept).Build(ctx, t)
	resp := testHttp.Do(t, req)
	require.Equal(t, wantCode, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()
	v, err := io.ReadAll(resp.Body)
	fmt.Printf("body %v\n", string(v))
	require.NoError(t, err)
	require.Equal(t, wantContentType, resp.Header.Get("Content-Type"))
	if wantContent {
		require.NotEmpty(t, v)
	} else {
		require.Empty(t, v)
	}
	gwShutdown()
}

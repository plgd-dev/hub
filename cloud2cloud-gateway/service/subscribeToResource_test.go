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

	router "github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/plgd-dev/cloud/authorization/provider"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certificateManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_SubscribeToResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	wantCode := http.StatusCreated
	wantContentType := message.AppJSON.String()
	wantContent := true
	wantEventType := events.EventType_ResourceChanged
	wantEventContent := map[interface{}]interface{}(map[interface{}]interface{}{"name": "Light", "power": uint64(0), "state": false})
	eventType := events.EventType_ResourceChanged
	uri := "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/light/1/subscriptions"
	accept := message.AppJSON.String()

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, provider.UserToken)

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer conn.Close()
	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	var listen certificateManager.Config
	err = envconfig.Process("LISTEN", &listen)
	require.NoError(t, err)

	listenCertManager, err := certificateManager.NewCertificateManager(listen)
	require.NoError(t, err)
	cfg := listenCertManager.GetServerTLSConfig()
	cfg.ClientAuth = tls.NoClientCert
	eventsServer, err := tls.Listen("tcp", "localhost:", cfg)
	require.NoError(t, err)

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
			defer r.Body.Close()
			assert.Equal(t, wantEventType, h.EventType)
			buf, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			var v interface{}
			err = json.Decode(buf, &v)
			assert.NoError(t, err)
			assert.Equal(t, wantEventContent, v)
			w.WriteHeader(http.StatusOK)
			eventsServer.Close()
		})).Methods("POST")
		http.Serve(eventsServer, r)
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
	req := test.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(provider.UserToken).AddHeader("Accept", accept).Build(ctx, t)
	resp := test.DoHTTPRequest(t, req)
	assert.Equal(t, wantCode, resp.StatusCode)
	defer resp.Body.Close()
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

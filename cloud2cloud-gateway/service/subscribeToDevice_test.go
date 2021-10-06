package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/v2/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/cloud/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/cloud/v2/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/v2/test"
	testCfg "github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/sdk/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandler_SubscribeToDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	wantCode := http.StatusCreated
	wantContentType := message.AppJSON.String()
	wantContent := true
	wantEventType := events.EventType_ResourcesPublished
	wantEventContent := test.ResourceLinksToResources(deviceID, test.TestDevsimResources)
	eventType := events.EventType_ResourcesPublished
	uri := "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/subscriptions"
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
			var v schema.ResourceLinks
			err = json.Decode(buf, &v)
			assert.NoError(t, err)
			links := make([]*commands.Resource, 0)
			for _, l := range v {
				pl := commands.SchemaResourceLinkToResource(l, time.Time{})
				pl.Href = "/" + strings.Join(strings.Split(pl.GetHref(), "/")[2:], "/")
				links = append(links, pl)
			}
			test.CleanUpResourcesArray(links)

			assert.Equal(t, wantEventContent, links)
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
	req := test.NewHTTPRequest(http.MethodPost, uri, bytes.NewBuffer(data)).AuthToken(token).Accept(accept).Build(ctx, t)
	resp := test.DoHTTPRequest(t, req)
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

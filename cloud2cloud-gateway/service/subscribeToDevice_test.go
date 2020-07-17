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

	"github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/uri"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/cloud/test"
	testCfg "github.com/go-ocf/cloud/test/config"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/sdk/schema"
	router "github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
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
	wantEventContent := test.ResourceLinksToPb(deviceID, test.GetAllBackendResourceLinks())
	eventType := events.EventType_ResourcesPublished
	uri := "https://" + testCfg.C2C_GW_HOST + uri.Devices + "/" + deviceID + "/subscriptions"
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
	shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	var listen certManager.Config
	err = envconfig.Process("LISTEN", &listen)
	require.NoError(t, err)

	listenCertManager, err := certManager.NewCertManager(listen)
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
			var v schema.ResourceLinks
			err = json.Decode(buf, &v)
			assert.NoError(t, err)
			links := make([]*pb.ResourceLink, 0, 32)
			for _, l := range v {
				pl := pb.SchemaResourceLinkToProto(l)
				pl.InstanceId = 0
				pl.Href = "/" + strings.Join(strings.Split(pl.GetHref(), "/")[2:], "/")
				links = append(links, &pl)
			}
			assert.Equal(t, test.SortResources(wantEventContent), test.SortResources(links))
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

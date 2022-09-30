package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	c2cEvents "github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	c2cTest "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/test"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerRetrieveDeviceSubscription(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.COAPS_TCP_SCHEME+testCfg.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Device)
	subID := subscriber.Subscribe(t, ctx, token, deviceID, "", c2cEvents.EventTypes{
		c2cEvents.EventType_ResourcesPublished,
		c2cEvents.EventType_ResourcesUnpublished,
	})
	require.NotEmpty(t, subID)

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.NotEmpty(t, events)

	const textPlain = "text/plain"
	type args struct {
		deviceID string
		subID    string
		token    string
	}
	tests := []struct {
		name            string
		args            args
		wantContentType string
		wantCode        int
		want            interface{}
	}{
		{
			name: "missing token",
			args: args{
				deviceID: deviceID,
				subID:    subID,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				deviceID: deviceID,
				subID:    subID,
				token:    oauthTest.GetAccessToken(t, testCfg.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired),
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token is expired",
		},
		{
			name: "invalid deviceID",
			args: args{
				deviceID: "invalidDeviceID",
				subID:    subID,
				token:    token,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "inaccessible uri",
		},
		{
			name: "invalid subID",
			args: args{
				deviceID: deviceID,
				subID:    "invalidSubID",
				token:    token,
			},
			wantCode:        http.StatusNotFound,
			wantContentType: textPlain,
			want:            "cannot retrieve device subscription: not found",
		},
		{
			name: "valid subscription",
			args: args{
				deviceID: deviceID,
				subID:    subID,
				token:    token,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				test.FieldJsonTag(c2cEvents.SubscriptionResponse{}, "SubscriptionID"): subID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := testHttp.NewHTTPRequest(http.MethodGet, c2cTest.C2CURI(uri.DeviceSubscription), nil).AuthToken(tt.args.token)
			rb.DeviceId(tt.args.deviceID).SubscriptionID(tt.args.subID)
			resp := testHttp.DoHTTPRequest(t, rb.Build(ctx, t))
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}

	subscriber.Unsubscribe(t, ctx, token, deviceID, "", subID)
	ev := <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

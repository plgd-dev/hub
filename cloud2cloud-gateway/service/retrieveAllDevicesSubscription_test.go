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
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerRetrieveDevicesSubscription(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	conn, err := grpc.Dial(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Devices)
	subID := subscriber.Subscribe(ctx, t, token, "", "", c2cEvents.EventTypes{
		c2cEvents.EventType_DevicesRegistered,
		c2cEvents.EventType_DevicesUnregistered, c2cEvents.EventType_DevicesOnline, c2cEvents.EventType_DevicesOffline,
	})
	require.NotEmpty(t, subID)

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.NotEmpty(t, events)

	const textPlain = "text/plain"
	type args struct {
		subID string
		token string
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
				subID: subID,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				subID: subID,
				token: oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired),
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token is expired",
		},
		{
			name: "invalid subID format",
			args: args{
				subID: "invalidSubID",
				token: token,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: inaccessible uri",
		},
		{
			name: "non-existing subID",
			args: args{
				subID: c2cTest.GetUniqueSubscriptionID(subID),
				token: token,
			},
			wantCode:        http.StatusNotFound,
			wantContentType: textPlain,
			want:            "cannot retrieve all devices subscription: not found",
		},
		{
			name: "valid subscription",
			args: args{
				subID: subID,
				token: token,
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
			rb := testHttp.NewHTTPRequest(http.MethodGet, c2cTest.C2CURI(uri.DevicesSubscription), nil).AuthToken(tt.args.token)
			rb.SubscriptionID(tt.args.subID)
			resp := testHttp.DoHTTPRequest(t, rb.Build(ctx, t))
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			var got interface{}
			testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType, &got)
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}

	subscriber.Unsubscribe(ctx, t, token, "", "", subID)
	ev := <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

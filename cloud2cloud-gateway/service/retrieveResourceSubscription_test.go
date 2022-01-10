package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	c2cEvents "github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	c2cService "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/service"
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

func TestRequestHandlerRetrieveResourceSubscription(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	// log.Setup(log.Config{Debug: true})

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)
	defer func() {
		_ = conn.Close()
	}()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()
	const switchID = "1"
	test.AddDeviceSwitchResources(ctx, t, deviceID, c, switchID)

	const eventsURI = "/events"
	eventsServer := c2cTest.NewEventsServer(t, eventsURI)
	defer eventsServer.Close(t)
	dataChan := eventsServer.Run(t)

	const resourceHref = test.TestResourceSwitchesHref
	subscriber := c2cTest.NewC2CSubscriber(eventsServer.GetPort(t), eventsURI, c2cTest.SubscriptionType_Resource)
	subID := subscriber.Subscribe(t, ctx, token, deviceID, resourceHref, c2cEvents.EventTypes{c2cEvents.EventType_ResourceChanged})
	require.NotEmpty(t, subID)

	events := c2cTest.WaitForEvents(dataChan, 3*time.Second)
	require.NotEmpty(t, events)

	const textPlain = "text/plain"
	type args struct {
		deviceID     string
		resourceHref string
		subID        string
		token        string
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
				deviceID:     deviceID,
				resourceHref: resourceHref,
				subID:        subID,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token contains an invalid number of segments",
		},
		{
			name: "expired token",
			args: args{
				deviceID:     deviceID,
				resourceHref: resourceHref,
				subID:        subID,
				token:        oauthTest.GetAccessToken(t, config.OAUTH_SERVER_HOST, oauthTest.ClientTestExpired),
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "invalid token: could not parse token: token is expired",
		},
		{
			name: "invalid deviceID",
			args: args{
				deviceID:     "invalidDeviceID",
				resourceHref: resourceHref,
				subID:        subID,
				token:        token,
			},
			wantCode:        http.StatusUnauthorized,
			wantContentType: textPlain,
			want:            "inaccessible uri",
		},
		{
			name: "invalid resourceHref",
			args: args{
				deviceID:     deviceID,
				resourceHref: "/invalidHref",
				subID:        subID,
				token:        token,
			},
			wantCode:        http.StatusBadRequest,
			wantContentType: textPlain,
			want:            "cannot retrieve resource subscription: invalid resource(/invalidHref)",
		},
		{
			name: "invalid subID",
			args: args{
				deviceID:     deviceID,
				resourceHref: resourceHref,
				subID:        "invalidSubID",
				token:        token,
			},
			wantCode:        http.StatusNotFound,
			wantContentType: textPlain,
			want:            "cannot retrieve resource subscription: not found",
		},
		{
			name: "valid subscription",
			args: args{
				deviceID:     deviceID,
				resourceHref: resourceHref,
				subID:        subID,
				token:        token,
			},
			wantCode:        http.StatusOK,
			wantContentType: message.AppJSON.String(),
			want: map[interface{}]interface{}{
				test.FieldJsonTag(c2cService.SubscriptionResponse{}, "SubscriptionID"): subID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := testHttp.NewHTTPRequest(http.MethodGet, c2cTest.C2CURI(uri.ResourceSubscription), nil).AuthToken(tt.args.token)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.resourceHref).SubscriptionID(tt.args.subID)
			resp := testHttp.DoHTTPRequest(t, rb.Build(ctx, t))
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"))
			got := testHttp.ReadHTTPResponse(t, resp.Body, tt.wantContentType)
			if tt.wantContentType == textPlain {
				require.Contains(t, got, tt.want)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}

	subscriber.Unsubscribe(t, ctx, token, deviceID, resourceHref, subID)
	ev := <-dataChan
	assert.Equal(t, c2cEvents.EventType_SubscriptionCanceled, ev.GetHeader().EventType)
}

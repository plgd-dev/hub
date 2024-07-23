package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getAllEvents(ctx context.Context, t *testing.T, client pb.GrpcGatewayClient) []interface{} {
	events := make([]interface{}, 0, len(test.GetAllBackendResourceLinks()))
	c, err := client.GetEvents(ctx, &pb.GetEventsRequest{
		TimestampFilter: 0,
	})
	require.NoError(t, err)
	for {
		value, err := c.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		event := pbTest.GetWrappedEvent(value)
		require.NotNil(t, event)
		events = append(events, event)
	}
	return events
}

func TestRequestHandlerGetEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	beforeOnBoard := time.Now()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	allEvents := getAllEvents(ctx, t, c)
	require.NotNil(t, allEvents)

	type args struct {
		accept    string
		deviceID  string
		href      string
		timestamp time.Time
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantLen      int
		wantHTTPCode int
	}{
		{
			name: "All events",
			args: args{
				accept: pkgHttp.ApplicationProtoJsonContentType,
			},
			wantLen:      len(allEvents),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Timestamp filter (No events)",
			args: args{
				accept:    pkgHttp.ApplicationProtoJsonContentType,
				timestamp: time.Now(),
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Timestamp filter (All events)",
			args: args{
				accept:    pkgHttp.ApplicationProtoJsonContentType,
				timestamp: beforeOnBoard,
			},
			wantLen:      len(allEvents),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Device filter (Invalid device)",
			args: args{
				accept:   pkgHttp.ApplicationProtoJsonContentType,
				deviceID: "test",
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Device filter (All devices)",
			args: args{
				accept:    pkgHttp.ApplicationProtoJsonContentType,
				deviceID:  deviceID,
				timestamp: beforeOnBoard,
			},
			wantLen:      len(allEvents),
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Resource filter (Invalid href)",
			args: args{
				accept:   pkgHttp.ApplicationProtoJsonContentType,
				deviceID: deviceID,
				href:     "test",
			},
			wantHTTPCode: http.StatusOK,
		},
		{
			name: "Resource filter (First resource)",
			args: args{
				accept:   pkgHttp.ApplicationProtoJsonContentType,
				deviceID: deviceID,
				href:     test.GetAllBackendResourceLinks()[0].Href,
			},
			wantLen:      1,
			wantHTTPCode: http.StatusOK,
		},
	}

	getURL := func(deviceID, href string) string {
		if deviceID != "" {
			if href != "" {
				return uri.AliasResourceEvents
			}
			return uri.AliasDeviceEvents
		}
		return uri.Events
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := httpgwTest.NewRequest(http.MethodGet, getURL(tt.args.deviceID, tt.args.href), nil).AuthToken(token).Accept(tt.args.accept)
			rb.DeviceId(tt.args.deviceID).ResourceHref(tt.args.href).Timestamp(tt.args.timestamp)
			resp := httpgwTest.HTTPDo(t, rb.Build())
			defer func() {
				_ = resp.Body.Close()
			}()
			assert.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			values := make([]*pb.GetEventsResponse, 0, 1)
			for {
				var value pb.GetEventsResponse
				err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &value)
				if errors.Is(err, io.EOF) {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				values = append(values, &value)
			}
			require.Len(t, values, tt.wantLen)
			pbTest.CheckGetEventsResponse(t, values)
		})
	}
}

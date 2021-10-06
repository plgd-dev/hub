package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/v2/http-gateway/test"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/test"
	"github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
	pbTest "github.com/plgd-dev/cloud/v2/test/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getAllEvents(t *testing.T, client pb.GrpcGatewayClient, ctx context.Context) []interface{} {
	events := make([]interface{}, 0, len(test.GetAllBackendResourceLinks()))
	c, err := client.GetEvents(ctx, &pb.GetEventsRequest{
		TimestampFilter: 0,
	})
	require.NoError(t, err)
	for {
		value, err := c.Recv()
		if err == io.EOF {
			break
		}
		event := pbTest.GetWrappedEvent(value)
		require.NotNil(t, event)
		events = append(events, event)
	}
	return events
}

func TestRequestHandler_getEvents(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := test.SetUp(ctx, t)
	defer tearDown()

	shutdownHttp := httpgwTest.SetUp(t)
	defer shutdownHttp()

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	beforeOnBoard := time.Now()
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	allEvents := getAllEvents(t, c, ctx)
	require.NotNil(t, allEvents)

	type args struct {
		accept    string
		deviceId  string
		href      string
		timestamp time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantLen int
	}{
		{
			name: "All events",
			args: args{
				accept: uri.ApplicationProtoJsonContentType,
			},
			wantErr: false,
			wantLen: len(allEvents),
		},
		{
			name: "Timestamp filter (No events)",
			args: args{
				accept:    uri.ApplicationProtoJsonContentType,
				timestamp: time.Now(),
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "Timestamp filter (All events)",
			args: args{
				accept:    uri.ApplicationProtoJsonContentType,
				timestamp: beforeOnBoard,
			},
			wantErr: false,
			wantLen: len(allEvents),
		},
		{
			name: "Device filter (Invalid device)",
			args: args{
				accept:   uri.ApplicationProtoJsonContentType,
				deviceId: "test",
			},
			wantErr: true,
		},
		{
			name: "Device filter (All devices)",
			args: args{
				accept:    uri.ApplicationProtoJsonContentType,
				deviceId:  deviceID,
				timestamp: beforeOnBoard,
			},
			wantErr: false,
			wantLen: len(allEvents),
		},
		{
			name: "Resource filter (Invalid href)",
			args: args{
				accept:   uri.ApplicationProtoJsonContentType,
				deviceId: deviceID,
				href:     "test",
			},
			wantErr: true,
		},
		{
			name: "Resource filter (First resource)",
			args: args{
				accept:   uri.ApplicationProtoJsonContentType,
				deviceId: deviceID,
				href:     test.GetAllBackendResourceLinks()[0].Href,
			},
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := uri.Events
			if tt.args.deviceId != "" {
				if tt.args.href != "" {
					url = uri.AliasResourceEvents
				} else {
					url = uri.AliasDeviceEvents
				}
			}
			request_builder := httpgwTest.NewRequest(http.MethodGet, url, nil).AuthToken(token)
			request_builder.Accept(tt.args.accept).DeviceId(tt.args.deviceId).ResourceHref(tt.args.href).Timestamp(tt.args.timestamp)
			request := request_builder.Build()
			trans := http.DefaultTransport.(*http.Transport).Clone()
			trans.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
			c := http.Client{
				Transport: trans,
			}
			resp, err := c.Do(request)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			values := make([]*pb.GetEventsResponse, 0, 1)
			for {
				var value pb.GetEventsResponse
				err = Unmarshal(resp.StatusCode, resp.Body, &value)
				if err == io.EOF {
					break
				}
				if tt.wantErr {
					require.Error(t, err)
					return
				} else {
					require.NoError(t, err)
					values = append(values, &value)
				}
			}
			require.Len(t, values, tt.wantLen)
			pbTest.CheckGetEventsResponse(t, deviceID, values)
		})
	}
}

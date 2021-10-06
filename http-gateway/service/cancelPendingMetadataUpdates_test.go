package service_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/cloud/v2/http-gateway/test"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	"github.com/plgd-dev/cloud/v2/test/config"
	oauthTest "github.com/plgd-dev/cloud/v2/test/oauth-server/test"
)

func TestRequestHandler_CancelPendingMetadataUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	_, devicePendings, shutdown := initPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(devicePendings), 2)

	type args struct {
		deviceID            string
		correlationIdFilter []string
		accept              string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *pb.CancelPendingCommandsResponse
	}{
		{
			name: "cancel one pending",
			args: args{
				deviceID:            devicePendings[0].DeviceID,
				correlationIdFilter: []string{devicePendings[0].CorrelationID},
				accept:              uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{devicePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:            devicePendings[0].DeviceID,
				correlationIdFilter: []string{devicePendings[0].CorrelationID},
				accept:              uri.ApplicationProtoJsonContentType,
			},
			wantErr: true,
		},
		{
			name: "cancel all events",
			args: args{
				deviceID: devicePendings[0].DeviceID,
				accept:   uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{devicePendings[1].CorrelationID},
			},
		},
	}

	token := oauthTest.GetDefaultServiceToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodDelete, uri.AliasDevicePendingMetadataUpdates, nil).AuthToken(token).Accept(tt.args.accept).DeviceId(tt.args.deviceID).AddCorrelantionIdFilter(tt.args.correlationIdFilter).Build()
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
			var v pb.CancelPendingCommandsResponse
			err = Unmarshal(resp.StatusCode, resp.Body, &v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			cmpCancel(t, tt.want, &v)
		})
	}
}

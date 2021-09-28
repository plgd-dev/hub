package service_test

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sort"
	"testing"
	"time"

	caService "github.com/plgd-dev/cloud/certificate-authority/test"
	coapgwTest "github.com/plgd-dev/cloud/coap-gateway/test"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	grpcgwService "github.com/plgd-dev/cloud/grpc-gateway/test"
	httpgwTest "github.com/plgd-dev/cloud/http-gateway/test"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	idService "github.com/plgd-dev/cloud/identity/test"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	raService "github.com/plgd-dev/cloud/resource-aggregate/test"
	rdService "github.com/plgd-dev/cloud/resource-directory/test"
	"github.com/plgd-dev/cloud/test"
	testCfg "github.com/plgd-dev/cloud/test/config"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type resourcePendingEvent struct {
	ResourceId    *commands.ResourceId
	CorrelationID string
}

type devicePendingEvent struct {
	DeviceID      string
	CorrelationID string
}

func initPendingEvents(ctx context.Context, t *testing.T) ([]resourcePendingEvent, []devicePendingEvent, func()) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	test.ClearDB(ctx, t)

	var closeFunc []func()

	oauthShutdown := oauthTest.SetUp(t)
	authShutdown := idService.SetUp(t)
	raShutdown := raService.SetUp(t)
	rdShutdown := rdService.SetUp(t)
	grpcShutdown := grpcgwService.SetUp(t)
	caShutdown := caService.SetUp(t)
	secureGWShutdown := coapgwTest.SetUp(t)

	closeFunc = append(closeFunc, caShutdown)
	closeFunc = append(closeFunc, grpcShutdown)
	closeFunc = append(closeFunc, rdShutdown)
	closeFunc = append(closeFunc, raShutdown)
	closeFunc = append(closeFunc, authShutdown)
	closeFunc = append(closeFunc, oauthShutdown)

	shutdownHttp := httpgwTest.SetUp(t)
	closeFunc = append(closeFunc, shutdownHttp)

	token := oauthTest.GetDefaultServiceToken(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	closeFunc = append(closeFunc, shutdownDevSim)

	secureGWShutdown()

	create := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.CreateResource(ctx, &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, "/light/1"),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	create()
	retrieve := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.GetResourceFromDevice(ctx, &pb.GetResourceFromDeviceRequest{
			ResourceId: commands.NewResourceID(deviceID, "/light/1"),
		})
		require.Error(t, err)
	}
	retrieve()
	update := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateResource(ctx, &pb.UpdateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, "/light/1"),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data: test.EncodeToCbor(t, map[string]interface{}{
					"power": 1,
				}),
			},
		})
		require.Error(t, err)
	}
	update()
	delete := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.DeleteResource(ctx, &pb.DeleteResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, "/light/1"),
		})
		require.Error(t, err)
	}
	delete()
	updateDeviceMetadata := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		_, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
			DeviceId:              deviceID,
			ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
		})
		require.Error(t, err)
	}
	updateDeviceMetadata()
	updateDeviceMetadata()

	client, err := c.GetPendingCommands(ctx, &pb.GetPendingCommandsRequest{})
	require.NoError(t, err)
	resourcePendings := make([]resourcePendingEvent, 0, 24)
	devicePendings := make([]devicePendingEvent, 0, 24)
	for {
		p, err := client.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		switch {
		case p.GetDeviceMetadataUpdatePending() != nil:
			v := p.GetDeviceMetadataUpdatePending()
			devicePendings = append(devicePendings, devicePendingEvent{
				DeviceID:      v.GetDeviceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceCreatePending() != nil:
			v := p.GetResourceCreatePending()
			resourcePendings = append(resourcePendings, resourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceRetrievePending() != nil:
			v := p.GetResourceRetrievePending()
			resourcePendings = append(resourcePendings, resourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceUpdatePending() != nil:
			v := p.GetResourceUpdatePending()
			resourcePendings = append(resourcePendings, resourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		case p.GetResourceDeletePending() != nil:
			v := p.GetResourceDeletePending()
			resourcePendings = append(resourcePendings, resourcePendingEvent{
				ResourceId:    v.GetResourceId(),
				CorrelationID: v.GetAuditContext().GetCorrelationId(),
			})
		}
	}

	return resourcePendings, devicePendings, func() {
		for i := range closeFunc {
			closeFunc[len(closeFunc)-1-i]()
		}
	}
}

func cmpCancel(t *testing.T, want *pb.CancelPendingCommandsResponse, got *pb.CancelPendingCommandsResponse) {
	sort.Strings(want.CorrelationIds)
	sort.Strings(got.CorrelationIds)
	require.Equal(t, want.CorrelationIds, got.CorrelationIds)
}

func TestRequestHandler_CancelPendingCommands(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	resourcePendings, _, shutdown := initPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(resourcePendings), 4)

	type args struct {
		deviceID            string
		href                string
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
				deviceID:            resourcePendings[0].ResourceId.GetDeviceId(),
				href:                resourcePendings[0].ResourceId.GetHref(),
				correlationIdFilter: []string{resourcePendings[0].CorrelationID},
				accept:              uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:            resourcePendings[0].ResourceId.GetDeviceId(),
				href:                resourcePendings[0].ResourceId.GetHref(),
				correlationIdFilter: []string{resourcePendings[0].CorrelationID},
				accept:              uri.ApplicationProtoJsonContentType,
			},
			wantErr: true,
		},
		{
			name: "cancel all events",
			args: args{
				deviceID: resourcePendings[0].ResourceId.GetDeviceId(),
				href:     resourcePendings[0].ResourceId.GetHref(),
				accept:   uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[1].CorrelationID, resourcePendings[2].CorrelationID, resourcePendings[3].CorrelationID},
			},
		},
	}

	token := oauthTest.GetDefaultServiceToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodDelete, uri.AliasResourcePendingCommands, nil).AuthToken(token).Accept(tt.args.accept).DeviceId(tt.args.deviceID).ResourceHref(tt.args.href).AddCorrelantionIdFilter(tt.args.correlationIdFilter).Build()
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

func TestRequestHandler_CancelResourceCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()

	resourcePendings, _, shutdown := initPendingEvents(ctx, t)
	defer shutdown()

	require.Equal(t, len(resourcePendings), 4)

	type args struct {
		deviceID      string
		href          string
		correlationId string
		accept        string
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
				deviceID:      resourcePendings[0].ResourceId.GetDeviceId(),
				href:          resourcePendings[0].ResourceId.GetHref(),
				correlationId: resourcePendings[0].CorrelationID,
				accept:        uri.ApplicationProtoJsonContentType,
			},
			want: &pb.CancelPendingCommandsResponse{
				CorrelationIds: []string{resourcePendings[0].CorrelationID},
			},
		},
		{
			name: "duplicate cancel event",
			args: args{
				deviceID:      resourcePendings[0].ResourceId.GetDeviceId(),
				href:          resourcePendings[0].ResourceId.GetHref(),
				correlationId: resourcePendings[0].CorrelationID,
				accept:        uri.ApplicationProtoJsonContentType,
			},
			wantErr: true,
		},
	}

	token := oauthTest.GetDefaultServiceToken(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodDelete, uri.AliasResourcePendingCommands+"/"+tt.args.correlationId, nil).AuthToken(token).Accept(tt.args.accept).DeviceId(tt.args.deviceID).ResourceHref(tt.args.href).Build()
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

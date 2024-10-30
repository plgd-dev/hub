package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

type contentChangedFilter struct {
	resourceChangedCh       chan eventbus.EventUnmarshaler
	deviceMetadataUpdatedCh chan *events.DeviceMetadataUpdated
}

func newContentChangedFilter() *contentChangedFilter {
	return &contentChangedFilter{
		resourceChangedCh:       make(chan eventbus.EventUnmarshaler, 2),
		deviceMetadataUpdatedCh: make(chan *events.DeviceMetadataUpdated, 3),
	}
}

func (f *contentChangedFilter) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		v, ok := iter.Next(ctx)
		if !ok {
			return nil
		}
		if v.EventType() == (&events.ResourceChanged{}).EventType() {
			select {
			case f.resourceChangedCh <- v:
			default:
			}
		}
		if v.EventType() == (&events.DeviceMetadataUpdated{}).EventType() {
			var ev events.DeviceMetadataUpdated
			err := v.Unmarshal(&ev)
			if err != nil {
				return err
			}
			select {
			case f.deviceMetadataUpdatedCh <- &ev:
			default:
			}
		}
	}
}

func (f *contentChangedFilter) WaitForResourceChanged(t time.Duration) eventbus.EventUnmarshaler {
	select {
	case v := <-f.resourceChangedCh:
		return v
	case <-time.After(t):
		return nil
	}
}

func (f *contentChangedFilter) WaitForDeviceMetadataUpdated(t time.Duration) *events.DeviceMetadataUpdated {
	select {
	case v := <-f.deviceMetadataUpdatedCh:
		return v
	case <-time.After(t):
		return nil
	}
}

func updateResource(t *testing.T, req *pb.UpdateResourceRequest, token string) error {
	const accept = pkgHttp.ApplicationProtoJsonContentType
	const contentType = pkgHttp.ApplicationProtoJsonContentType
	data, err := httpTest.GetContentData(req.GetContent(), contentType)
	if err != nil {
		return err
	}

	rb := httpgwTest.NewRequest(http.MethodPut, uri.AliasDeviceResource, bytes.NewReader(data)).AuthToken(token).Accept(accept).ContentType(contentType)
	rb.DeviceId(req.GetResourceId().GetDeviceId()).ResourceHref(req.GetResourceId().GetHref()).ResourceInterface(req.GetResourceInterface())
	resp := httpgwTest.HTTPDo(t, rb.Build())
	defer func() {
		_ = resp.Body.Close()
	}()

	var got pb.UpdateResourceResponse
	err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
	if err != nil {
		return err
	}
	return nil
}

func TestRequestHandlerUpdateDeviceMetadata(t *testing.T) {
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

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	logger := log.NewLogger(log.MakeDefaultConfig())

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	naClient, s, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), fileWatcher, logger, noop.NewTracerProvider(), subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer func() {
		s.Close()
		naClient.Close()
	}()
	tmp := uuid.New()
	v := newContentChangedFilter()
	obs, err := s.Subscribe(ctx, tmp.String(), utils.GetDeviceSubject("*", deviceID), v)
	require.NoError(t, err)
	defer func() {
		_ = obs.Close()
	}()

	updateDeviceTwinSynchronization := func(in *pb.UpdateDeviceMetadataRequest) error {
		data, errM := protojson.Marshal(in)
		require.NoError(t, errM)

		rb := httpgwTest.NewRequest(http.MethodPut, uri.DeviceMetadata, bytes.NewReader(data)).AuthToken(token).DeviceId(deviceID)
		resp := httpgwTest.HTTPDo(t, rb.Build())
		defer func(r *http.Response) {
			_ = r.Body.Close()
		}(resp)

		var got pb.UpdateDeviceMetadataResponse
		errM = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
		return errM
	}

	err = updateDeviceTwinSynchronization(&pb.UpdateDeviceMetadataRequest{
		DeviceId:    deviceID,
		TwinEnabled: false,
	})
	require.NoError(t, err)

	ev := v.WaitForDeviceMetadataUpdated(time.Second)
	require.NotEmpty(t, ev)
	require.False(t, ev.GetTwinEnabled())

	err = updateResource(t, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	}, token)
	require.NoError(t, err)
	err = updateResource(t, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	}, token)
	require.NoError(t, err)

	evResourceChanged := v.WaitForResourceChanged(time.Second)
	require.Empty(t, evResourceChanged)

	err = updateDeviceTwinSynchronization(&pb.UpdateDeviceMetadataRequest{
		DeviceId:    deviceID,
		TwinEnabled: true,
	})
	require.NoError(t, err)

	for {
		ev = v.WaitForDeviceMetadataUpdated(time.Second * 5)
		require.NotEmpty(t, ev)
		if ev.GetTwinEnabled() {
			break
		}
	}

	err = updateResource(t, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	}, token)
	require.NoError(t, err)
	err = updateResource(t, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	}, token)
	require.NoError(t, err)

	evResourceChanged = v.WaitForResourceChanged(time.Second)
	require.NotEmpty(t, evResourceChanged)
}

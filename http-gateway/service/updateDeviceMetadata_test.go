package service_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

type contentChangedFilter struct {
	resourceChangedCh       chan eventbus.EventUnmarshaler
	deviceMetadataUpdatedCh chan *events.DeviceMetadataUpdated
}

func NewContentChangedFilter() *contentChangedFilter {
	return &contentChangedFilter{
		resourceChangedCh:       make(chan eventbus.EventUnmarshaler, 2),
		deviceMetadataUpdatedCh: make(chan *events.DeviceMetadataUpdated, 1),
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
			if ev.GetShadowSynchronization() == commands.ShadowSynchronization_UNSET {
				continue
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

func updateResource(t *testing.T, ctx context.Context, req *pb.UpdateResourceRequest, token, accept, contentType string) (*events.ResourceUpdated, error) {
	data, err := httpgwTest.GetContentData(req.GetContent(), contentType)
	if err != nil {
		return nil, err
	}

	rb := httpgwTest.NewRequest(http.MethodPut, uri.AliasDeviceResource, bytes.NewReader([]byte(data))).AuthToken(token).Accept(accept).ContentType(contentType)
	rb.DeviceId(req.GetResourceId().GetDeviceId()).ResourceHref(req.GetResourceId().GetHref()).ResourceInterface(req.GetResourceInterface())
	resp := httpgwTest.HTTPDo(t, rb.Build())
	defer func() {
		_ = resp.Body.Close()
	}()

	var got pb.UpdateResourceResponse
	err = Unmarshal(resp.StatusCode, resp.Body, &got)
	if err != nil {
		return nil, err
	}
	return got.GetData(), nil
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
	ctx = kitNetGrpc.CtxWithToken(ctx, token)

	conn, err := grpc.Dial(config.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, config.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	logger, err := log.NewLogger(log.MakeDefaultConfig())
	require.NoError(t, err)
	naClient, s, err := natsTest.NewClientAndSubscriber(config.MakeSubscriberConfig(), logger, subscriber.WithUnmarshaler(utils.Unmarshal))
	require.NoError(t, err)
	defer func() {
		s.Close()
		naClient.Close()
	}()
	tmp := uuid.New()
	v := NewContentChangedFilter()
	obs, err := s.Subscribe(ctx, tmp.String(), utils.GetDeviceSubject("*", deviceID), v)
	require.NoError(t, err)
	defer func() {
		_ = obs.Close()
	}()

	updateDeviceShadowSynchronization := func(ctx context.Context, in *pb.UpdateDeviceMetadataRequest) (*pb.UpdateDeviceMetadataResponse, error) {
		data, err := protojson.Marshal(in)
		require.NoError(t, err)

		rb := httpgwTest.NewRequest(http.MethodPut, uri.DeviceMetadata, bytes.NewReader(data)).AuthToken(token).DeviceId(deviceID)
		resp := httpgwTest.HTTPDo(t, rb.Build())
		defer func() {
			_ = resp.Body.Close()
		}()

		var got pb.UpdateDeviceMetadataResponse
		err = Unmarshal(resp.StatusCode, resp.Body, &got)
		if err != nil {
			return nil, err
		}
		return &got, nil
	}

	_, err = updateDeviceShadowSynchronization(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
	})
	require.NoError(t, err)

	ev := v.WaitForDeviceMetadataUpdated(time.Second)
	require.NotEmpty(t, ev)
	require.Equal(t, commands.ShadowSynchronization_DISABLED, ev.GetShadowSynchronization())

	_, err = updateResource(t, ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	}, token, uri.ApplicationProtoJsonContentType, uri.ApplicationProtoJsonContentType)
	require.NoError(t, err)
	_, err = updateResource(t, ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	}, token, uri.ApplicationProtoJsonContentType, uri.ApplicationProtoJsonContentType)
	require.NoError(t, err)

	evResourceChanged := v.WaitForResourceChanged(time.Second)
	require.Empty(t, evResourceChanged)

	_, err = updateDeviceShadowSynchronization(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_ENABLED,
	})
	require.NoError(t, err)

	ev = v.WaitForDeviceMetadataUpdated(time.Second * 5)
	require.NotEmpty(t, ev)
	require.Equal(t, commands.ShadowSynchronization_ENABLED, ev.GetShadowSynchronization())

	_, err = updateResource(t, ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	}, token, uri.ApplicationProtoJsonContentType, uri.ApplicationProtoJsonContentType)
	require.NoError(t, err)
	_, err = updateResource(t, ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	}, token, uri.ApplicationProtoJsonContentType, uri.ApplicationProtoJsonContentType)
	require.NoError(t, err)

	evResourceChanged = v.WaitForResourceChanged(time.Second)
	require.NotEmpty(t, evResourceChanged)
}

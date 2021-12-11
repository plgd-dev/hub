package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/subscriber"
	natsTest "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	testCfg "github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type contentChangedFilter struct {
	resourceChangedCh chan eventbus.EventUnmarshaler
}

func NewContentChangedFilter() *contentChangedFilter {
	return &contentChangedFilter{
		resourceChangedCh: make(chan eventbus.EventUnmarshaler, 2),
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

func TestRequestHandler_UpdateDeviceMetadata(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), testCfg.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	deviceID, shutdownDevSim := test.OnboardDevSim(ctx, t, c, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	logger, err := log.NewLogger(log.Config{})
	require.NoError(t, err)
	naClient, s, err := natsTest.NewClientAndSubscriber(testCfg.MakeSubscriberConfig(), logger, subscriber.WithUnmarshaler(utils.Unmarshal))
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
		err := obs.Close()
		assert.NoError(t, err)
	}()

	_, err = c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
		TimeToLive:            int64(99 * time.Millisecond),
	})
	require.Error(t, err)

	ev, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_DISABLED,
	})
	require.NoError(t, err)
	require.Equal(t, commands.ShadowSynchronization_DISABLED, ev.GetData().GetShadowSynchronization())

	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	})
	require.NoError(t, err)
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	})
	require.NoError(t, err)

	evResourceChanged := v.WaitForResourceChanged(time.Second)
	require.Empty(t, evResourceChanged)

	ev, err = c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:              deviceID,
		ShadowSynchronization: pb.UpdateDeviceMetadataRequest_ENABLED,
	})
	require.NoError(t, err)

	require.Equal(t, commands.ShadowSynchronization_ENABLED, ev.GetData().GetShadowSynchronization())

	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 2,
			}),
		},
	})
	require.NoError(t, err)
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceInterface: interfaces.OC_IF_BASELINE,
		ResourceId:        commands.NewResourceID(deviceID, test.TestResourceLightInstanceHref("1")),
		Content: &pb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data: test.EncodeToCbor(t, map[string]interface{}{
				"power": 0,
			}),
		},
	})
	require.NoError(t, err)

	evResourceChanged = v.WaitForResourceChanged(time.Second)
	require.NotEmpty(t, evResourceChanged)
}

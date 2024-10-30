package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
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
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type contentChangedFilter struct {
	resourceChangedCh chan eventbus.EventUnmarshaler
}

func newContentChangedFilter() *contentChangedFilter {
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

type deviceMetadataUpdatedFilter struct {
	ch chan eventbus.EventUnmarshaler
}

func newDeviceMetadataUpdatedFilter() *deviceMetadataUpdatedFilter {
	return &deviceMetadataUpdatedFilter{
		ch: make(chan eventbus.EventUnmarshaler, 2),
	}
}

func (f *deviceMetadataUpdatedFilter) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		v, ok := iter.Next(ctx)
		if !ok {
			return nil
		}
		if v.EventType() == (&events.DeviceMetadataUpdated{}).EventType() {
			select {
			case f.ch <- v:
			default:
			}
		}
	}
}

func (f *deviceMetadataUpdatedFilter) WaitForEvent(t time.Duration, correlationID string) (*events.DeviceMetadataUpdated, error) {
	deadline := time.Now().Add(t)
	for {
		select {
		case v := <-f.ch:
			var ev events.DeviceMetadataUpdated
			err := v.Unmarshal(&ev)
			if err != nil {
				return nil, err
			}
			if correlationID == "" {
				return &ev, nil
			}
			if ev.GetAuditContext().GetCorrelationId() == correlationID {
				return &ev, nil
			}
		case <-time.After(time.Until(deadline)):
			return nil, errors.New("timeout")
		}
	}
}

func TestRequestHandlerUpdateDeviceMetadataTwinEnabled(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

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
	deviceMetadataUpdatedFilter := newDeviceMetadataUpdatedFilter()
	obs, err := s.Subscribe(ctx, tmp.String(), utils.GetDeviceSubject("*", deviceID), v)
	require.NoError(t, err)
	obsDeviceMetadataUpdated, err := s.Subscribe(ctx, uuid.New().String(), utils.GetDeviceMetadataEventSubject("*", deviceID, (&events.DeviceMetadataUpdated{}).EventType()), deviceMetadataUpdatedFilter)
	require.NoError(t, err)
	defer func() {
		errC := obs.Close()
		require.NoError(t, errC)
		errC = obsDeviceMetadataUpdated.Close()
		require.NoError(t, errC)
	}()

	_, err = c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:    deviceID,
		TwinEnabled: false,
		TimeToLive:  int64(99 * time.Millisecond),
	})
	require.Error(t, err)

	ev, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:    deviceID,
		TwinEnabled: false,
	})
	require.NoError(t, err)
	require.False(t, ev.GetData().GetTwinEnabled())
	require.Equal(t, commands.TwinSynchronization_DISABLED, ev.GetData().GetTwinSynchronization().GetState())

	deviceMetadataUpdated, err := deviceMetadataUpdatedFilter.WaitForEvent(time.Second, ev.GetData().GetAuditContext().GetCorrelationId())
	require.NoError(t, err)
	require.False(t, deviceMetadataUpdated.GetTwinEnabled())
	require.Equal(t, commands.TwinSynchronization_DISABLED, deviceMetadataUpdated.GetTwinSynchronization().GetState())

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
		DeviceId:    deviceID,
		TwinEnabled: true,
	})
	require.NoError(t, err)
	require.True(t, ev.GetData().GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, ev.GetData().GetTwinSynchronization().GetState())

	deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, ev.GetData().GetAuditContext().GetCorrelationId())
	require.NoError(t, err)
	require.True(t, deviceMetadataUpdated.GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, deviceMetadataUpdated.GetTwinSynchronization().GetState())

	for {
		deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, "")
		require.NoError(t, err)
		require.True(t, deviceMetadataUpdated.GetTwinEnabled())
		if deviceMetadataUpdated.GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC {
			break
		}
	}

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

func waitForResourceChanged(filter *contentChangedFilter, ignoreHrefs ...string) eventbus.EventUnmarshaler {
	ev := filter.WaitForResourceChanged(time.Second)
	if ev != nil {
		evChanged := events.ResourceChanged{}
		if err := ev.Unmarshal(&evChanged); err != nil {
			return ev
		}
		for _, ignoreHref := range ignoreHrefs {
			if ignoreHref == evChanged.GetResourceId().GetHref() {
				return waitForResourceChanged(filter, ignoreHrefs...)
			}
		}
	}
	return ev
}

func TestRequestHandlerUpdateDeviceMetadataTwinForceSynchronization(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

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
	deviceMetadataUpdatedFilter := newDeviceMetadataUpdatedFilter()
	obs, err := s.Subscribe(ctx, tmp.String(), utils.GetDeviceSubject("*", deviceID), v)
	require.NoError(t, err)
	obsDeviceMetadataUpdated, err := s.Subscribe(ctx, uuid.New().String(), utils.GetDeviceMetadataEventSubject("*", deviceID, (&events.DeviceMetadataUpdated{}).EventType()), deviceMetadataUpdatedFilter)
	require.NoError(t, err)
	defer func() {
		errC := obs.Close()
		require.NoError(t, errC)
		errC = obsDeviceMetadataUpdated.Close()
		require.NoError(t, errC)
	}()

	ev, err := c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:    deviceID,
		TwinEnabled: false,
	})
	require.NoError(t, err)
	require.False(t, ev.GetData().GetTwinEnabled())
	require.Equal(t, commands.TwinSynchronization_DISABLED, ev.GetData().GetTwinSynchronization().GetState())

	deviceMetadataUpdated, err := deviceMetadataUpdatedFilter.WaitForEvent(time.Second, ev.GetData().GetAuditContext().GetCorrelationId())
	require.NoError(t, err)
	require.False(t, deviceMetadataUpdated.GetTwinEnabled())
	require.Equal(t, commands.TwinSynchronization_DISABLED, deviceMetadataUpdated.GetTwinSynchronization().GetState())
	require.Equal(t, int64(0), ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt())

	evResourceChanged := waitForResourceChanged(v, plgdtime.ResourceURI)
	require.Empty(t, evResourceChanged)

	// TwinForceSynchronization - enable twin
	checkTwin := time.Now().UnixNano()
	ev, err = c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:                 deviceID,
		TwinForceSynchronization: true,
	})
	require.NoError(t, err)
	require.True(t, ev.GetData().GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, ev.GetData().GetTwinSynchronization().GetState())

	deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, ev.GetData().GetAuditContext().GetCorrelationId())
	require.NoError(t, err)
	require.True(t, deviceMetadataUpdated.GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, deviceMetadataUpdated.GetTwinSynchronization().GetState())
	require.Greater(t, ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt(), checkTwin)
	checkTwin = ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt()

	for {
		deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, "")
		require.NoError(t, err)
		require.True(t, deviceMetadataUpdated.GetTwinEnabled())
		require.Equal(t, checkTwin, ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt())
		if deviceMetadataUpdated.GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC {
			break
		}
	}

	// update resource
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
	evResourceChanged = waitForResourceChanged(v, plgdtime.ResourceURI)
	require.NotEmpty(t, evResourceChanged)

	// revert update resource
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
	evResourceChanged = waitForResourceChanged(v, plgdtime.ResourceURI)
	require.NotEmpty(t, evResourceChanged)

	// TwinForceSynchronization - twin is already enabled
	checkTwin = time.Now().UnixNano()
	ev, err = c.UpdateDeviceMetadata(ctx, &pb.UpdateDeviceMetadataRequest{
		DeviceId:                 deviceID,
		TwinForceSynchronization: true,
	})
	require.NoError(t, err)
	require.True(t, ev.GetData().GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, ev.GetData().GetTwinSynchronization().GetState())

	deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, ev.GetData().GetAuditContext().GetCorrelationId())
	require.NoError(t, err)
	require.True(t, deviceMetadataUpdated.GetTwinEnabled())
	require.NotEqual(t, commands.TwinSynchronization_DISABLED, deviceMetadataUpdated.GetTwinSynchronization().GetState())
	require.Greater(t, ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt(), checkTwin)
	checkTwin = ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt()

	evResourceChanged = waitForResourceChanged(v, plgdtime.ResourceURI)
	require.Empty(t, evResourceChanged)

	for {
		deviceMetadataUpdated, err = deviceMetadataUpdatedFilter.WaitForEvent(time.Second, "")
		require.NoError(t, err)
		require.True(t, deviceMetadataUpdated.GetTwinEnabled())
		require.Equal(t, checkTwin, ev.GetData().GetTwinSynchronization().GetForceSynchronizationAt())
		if deviceMetadataUpdated.GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC {
			break
		}
	}
}

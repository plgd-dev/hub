package test

import (
	"context"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device"
	"github.com/plgd-dev/hub/v2/test/device/ocf"
	hubTestPb "github.com/plgd-dev/hub/v2/test/pb"
	"github.com/plgd-dev/hub/v2/test/sdk"
	"github.com/stretchr/testify/require"
)

func SubscribeToEvents(t *testing.T, subClient pb.GrpcGateway_SubscribeToEventsClient, sub *pb.SubscribeToEvents) (string, string) {
	err := subClient.Send(sub)
	require.NoError(t, err)
	ev, err := subClient.Recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		CorrelationId:  ev.GetCorrelationId(),
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	hubTest.CheckProtobufs(t, expectedEvent, ev, hubTest.RequireToCheckFunc(require.Equal))
	return ev.GetSubscriptionId(), ev.GetCorrelationId()
}

func SubscribeToAllEvents(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, correllationID string) (pb.GrpcGateway_SubscribeToEventsClient, string) {
	subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
	require.NoError(t, err)
	err = subClient.Send(&pb.SubscribeToEvents{
		CorrelationId: correllationID,
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{},
		},
	})
	require.NoError(t, err)
	ev, err := subClient.Recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.GetSubscriptionId(),
		CorrelationId:  correllationID,
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	hubTest.CheckProtobufs(t, expectedEvent, ev, hubTest.RequireToCheckFunc(require.Equal))
	return subClient, ev.GetSubscriptionId()
}

func waitForEvents(t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, events map[string]*pb.Event) error {
	for {
		ev, err := client.Recv()
		if err != nil {
			return err
		}
		expectedEvent, ok := events[hubTestPb.GetEventID(ev)]
		if !ok {
			t.Logf("unexpected event %+v", ev)
			continue
		}
		hubTestPb.CmpEvent(t, expectedEvent, ev, "")
		delete(events, hubTestPb.GetEventID(ev))
		if len(events) == 0 {
			return nil
		}
	}
}

func WaitForDeviceStatus(t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, deviceID, subID, correlationID string, status commands.Connection_Status) error {
	var dmus []*pb.Event_DeviceMetadataUpdated

	dmus = append(dmus, &pb.Event_DeviceMetadataUpdated{
		DeviceMetadataUpdated: hubTestPb.MakeDeviceMetadataUpdated(deviceID, status, hubTest.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_OUT_OF_SYNC, ""),
	})
	if status == commands.Connection_ONLINE {
		dmus = append(dmus, &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: hubTestPb.MakeDeviceMetadataUpdated(deviceID, status, hubTest.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_SYNCING, ""),
		})
		dmus = append(dmus, &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: hubTestPb.MakeDeviceMetadataUpdated(deviceID, status, hubTest.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME), true, commands.TwinSynchronization_IN_SYNC, ""),
		})
	}

	expectedEvents := make(map[string]*pb.Event)
	for _, dmu := range dmus {
		expectedEvents[hubTestPb.GetEventID(&pb.Event{Type: dmu})] = &pb.Event{
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type:           dmu,
		}
	}
	return waitForEvents(t, client, expectedEvents)
}

func WaitForRegistered(t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, deviceID, subID, correlationID string) error {
	dr := &pb.Event_DeviceRegistered_{
		DeviceRegistered: &pb.Event_DeviceRegistered{
			DeviceIds: []string{deviceID},
			EventMetadata: &isEvents.EventMetadata{
				HubId: config.HubID(),
			},
		},
	}
	expectedEvents := map[string]*pb.Event{
		hubTestPb.GetEventID(&pb.Event{Type: dr}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type:           dr,
		},
	}
	return waitForEvents(t, client, expectedEvents)
}

func ForceReprovision(ctx context.Context, c pb.GrpcGatewayClient, deviceID string) error {
	data, err := ToJSON(ResourcePlgdDps{ForceReprovision: true})
	if err != nil {
		return err
	}
	_, err = c.UpdateResource(ctx, &pb.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, ResourcePlgdDpsHref),
		Content: &pb.Content{
			ContentType: message.AppJSON.String(),
			Data:        data,
		},
	})
	return err
}

type configureDpsDevice = func(ctx context.Context, deviceID string, client *deviceClient.Client) error

// onboardConfig represents the configuration options available for onboarding.
type onboardConfig struct {
	sdkID     string
	sdkRootCA struct {
		certificate []byte
		key         []byte
	}
	validFrom       string
	validFor        string
	configureDevice configureDpsDevice
}

// Option interface used for setting optional onboardConfig properties.
type Option interface {
	apply(*onboardConfig)
}

type optionFunc func(*onboardConfig)

func (o optionFunc) apply(c *onboardConfig) {
	o(c)
}

// WithSdkID creates Option that overrides the default device ID used by the SDK client.
func WithSdkID(id string) Option {
	return optionFunc(func(cfg *onboardConfig) {
		cfg.sdkID = id
	})
}

// WithSdkRootCA creates Option that overrides the certificate authority used by the SDK client.
func WithSdkRootCA(certificate, key []byte) Option {
	return optionFunc(func(cfg *onboardConfig) {
		cfg.sdkRootCA.certificate = certificate
		cfg.sdkRootCA.key = key
	})
}

// WithValidity creates Option that overrides the ValidFrom timestamp and CertExpiry
// interval used by the SDK client when generating certificates.
func WithValidity(validFrom, validFor string) Option {
	return optionFunc(func(cfg *onboardConfig) {
		cfg.validFrom = validFrom
		cfg.validFor = validFor
	})
}

// WithConfigureDevice sets a function that will be called after the device is owned, but before the device is onboarded.
func WithConfigureDevice(configureDevice configureDpsDevice) Option {
	return optionFunc(func(cfg *onboardConfig) {
		cfg.configureDevice = configureDevice
	})
}

func newOnboardConfig(opts ...Option) *onboardConfig {
	cfg := &onboardConfig{
		sdkID:     isEvents.OwnerToUUID(DPSOwner),
		validFrom: "2000-01-01T12:00:00Z",
		validFor:  "876000h", // 100 years
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

func (cfg *onboardConfig) sdkOptions() []sdk.Option {
	var sdkOptions []sdk.Option
	sdkOptions = append(sdkOptions, sdk.WithID(cfg.sdkID))
	if cfg.sdkRootCA.certificate != nil && cfg.sdkRootCA.key != nil {
		sdkOptions = append(sdkOptions, sdk.WithRootCA(cfg.sdkRootCA.certificate, cfg.sdkRootCA.key))
	}
	if cfg.validFrom != "" {
		sdkOptions = append(sdkOptions, sdk.WithValidity(cfg.validFrom, cfg.validFor))
	}
	return sdkOptions
}

func OnboardDpsSim(ctx context.Context, t *testing.T, gc pb.GrpcGatewayClient, deviceID, dpsEndpoint string, expectedResources []schema.ResourceLink, opts ...Option) (string, func()) {
	d := ocf.NewDevice(deviceID, TestDeviceName)
	cleanup := OnboardDpsSimDevice(ctx, t, gc, d, dpsEndpoint, expectedResources, opts...)
	return d.GetID(), cleanup
}

func OnboardDpsSimDevice(ctx context.Context, t *testing.T, gc pb.GrpcGatewayClient, d device.Device, dpsEndpoint string, expectedResources []schema.ResourceLink, opts ...Option) func() {
	onboardCfg := newOnboardConfig(opts...)
	sdkOptions := onboardCfg.sdkOptions()
	cloudSID := config.HubID()
	require.NotEmpty(t, cloudSID)
	devClient, err := sdk.NewClient(sdkOptions...)
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	deviceID, err := devClient.OwnDevice(ctx, d.GetID(), deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)
	d.SetID(deviceID)

	if onboardCfg.configureDevice != nil {
		err = onboardCfg.configureDevice(ctx, d.GetID(), devClient)
		require.NoError(t, err)
	}

	onboard := func() {
		var v interface{}
		endpoint := config.ACTIVE_DPS_SCHEME + "://" + dpsEndpoint
		err = devClient.UpdateResource(ctx, d.GetID(), ResourcePlgdDpsHref, ResourcePlgdDps{Endpoint: &endpoint}, &v)
		require.NoError(t, err)
	}
	if len(expectedResources) > 0 {
		subClient, err := client.New(gc).GrpcGatewayClient().SubscribeToEvents(ctx)
		require.NoError(t, err)
		subID, corID := SubscribeToEvents(t, subClient, &pb.SubscribeToEvents{
			CorrelationId: "allEvents",
			Action: &pb.SubscribeToEvents_CreateSubscription_{
				CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{},
			},
		})
		onboard()
		hubTest.WaitForDevice(t, subClient, d, subID, corID, expectedResources)
		err = subClient.CloseSend()
		require.NoError(t, err)
	} else {
		onboard()
	}

	return func() {
		client, err := sdk.NewClient(sdkOptions...)
		require.NoError(t, err)
		err = client.DisownDevice(ctx, d.GetID())
		require.NoError(t, err)
		err = client.Close(ctx)
		require.NoError(t, err)
		time.Sleep(time.Second * 2)
	}
}

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

// DeviceMetadataUpdatedHandler handler of events.
type DeviceMetadataUpdatedHandler = interface {
	HandleDeviceMetadataUpdated(ctx context.Context, val *events.DeviceMetadataUpdated) error
}

// DeviceRegisteredHandler handler of events.
type DeviceRegisteredHandler = interface {
	HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error
}

// DeviceUnregisteredHandler handler of events.
type DeviceUnregisteredHandler = interface {
	HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error
}

// DevicesSubscription subscription.
type DevicesSubscription struct {
	client                       pb.GrpcGateway_SubscribeToEventsClient
	subscriptionID               string
	deviceMetadataUpdatedHandler DeviceMetadataUpdatedHandler
	deviceRegisteredHandler      DeviceRegisteredHandler
	deviceUnregisteredHandler    DeviceUnregisteredHandler
	closeErrorHandler            SubscriptionHandler

	wait     func()
	canceled uint32
}

// NewDevicesSubscription creates new devices subscriptions to listen events: device online, device offline, device registered, device unregistered.
// JWT token must be stored in context for grpc call.
func (c *Client) NewDevicesSubscription(ctx context.Context, handle SubscriptionHandler) (*DevicesSubscription, error) {
	return NewDevicesSubscription(ctx, handle, handle, c.gateway)
}

// NewDevicesSubscription creates new devices subscriptions to listen events: device online, device offline, device registered, device unregistered.
// JWT token must be stored in context for grpc call.
func NewDevicesSubscription(ctx context.Context, closeErrorHandler SubscriptionHandler, handle interface{}, gwClient pb.GrpcGatewayClient) (*DevicesSubscription, error) {
	var deviceMetadataUpdatedHandler DeviceMetadataUpdatedHandler
	var deviceRegisteredHandler DeviceRegisteredHandler
	var deviceUnregisteredHandler DeviceUnregisteredHandler
	filterEvents := make([]pb.SubscribeToEvents_CreateSubscription_Event, 0, 1)
	if v, ok := handle.(DeviceMetadataUpdatedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED)
		deviceMetadataUpdatedHandler = v
	}
	if v, ok := handle.(DeviceRegisteredHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_REGISTERED)
		deviceRegisteredHandler = v
	}
	if v, ok := handle.(DeviceUnregisteredHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeToEvents_CreateSubscription_UNREGISTERED)
		deviceUnregisteredHandler = v
	}

	if deviceMetadataUpdatedHandler == nil && deviceRegisteredHandler == nil && deviceUnregisteredHandler == nil {
		return nil, errors.New("invalid handler - it's supports: DeviceMetadataUpdatedHandler, DeviceRegisteredHandler, DeviceUnregisteredHandler")
	}
	client, err := New(gwClient).SubscribeToEventsWithCurrentState(ctx, time.Minute)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: filterEvents,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	ev, err := client.Recv()
	if err != nil {
		return nil, err
	}
	op := ev.GetOperationProcessed()
	if op == nil {
		return nil, fmt.Errorf("unexpected event %+v", ev)
	}
	if op.GetErrorStatus().GetCode() != pb.Event_OperationProcessed_ErrorStatus_OK {
		return nil, errors.New(op.GetErrorStatus().GetMessage())
	}

	var wg sync.WaitGroup
	sub := &DevicesSubscription{
		client:                       client,
		closeErrorHandler:            closeErrorHandler,
		subscriptionID:               ev.GetSubscriptionId(),
		deviceMetadataUpdatedHandler: deviceMetadataUpdatedHandler,
		deviceRegisteredHandler:      deviceRegisteredHandler,
		deviceUnregisteredHandler:    deviceUnregisteredHandler,
		wait:                         wg.Wait,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sub.runRecv()
	}()

	return sub, nil
}

// Cancel cancels subscription.
func (s *DevicesSubscription) Cancel() (wait func(), err error) {
	if !atomic.CompareAndSwapUint32(&s.canceled, 0, 1) {
		return s.wait, nil
	}
	err = s.client.CloseSend()
	if err != nil {
		return nil, err
	}
	return s.wait, nil
}

// ID returns subscription id.
func (s *DevicesSubscription) ID() string {
	return s.subscriptionID
}

func (s *DevicesSubscription) handleEvent(e *pb.Event) error {
	if ct := e.GetDeviceMetadataUpdated(); ct != nil {
		return s.deviceMetadataUpdatedHandler.HandleDeviceMetadataUpdated(s.client.Context(), ct)
	}

	if ct := e.GetDeviceRegistered(); ct != nil {
		return s.deviceRegisteredHandler.HandleDeviceRegistered(s.client.Context(), ct)
	}

	if ct := e.GetDeviceUnregistered(); ct != nil {
		return s.deviceUnregisteredHandler.HandleDeviceUnregistered(s.client.Context(), ct)
	}

	return fmt.Errorf("unknown event occurs %T on recv devices events: %+v", e, e)
}

func (s *DevicesSubscription) cancelAndHandlerError(err error) {
	var errors *multierror.Error
	if err != nil {
		errors = multierror.Append(errors, err)
	}
	if _, err := s.Cancel(); err != nil {
		errors = multierror.Append(errors, fmt.Errorf("failed to cancel device subscription: %w", err))
	}
	if errors.ErrorOrNil() != nil {
		s.closeErrorHandler.Error(errors)
	}
}

func (s *DevicesSubscription) handleCancel(cancel *pb.Event_SubscriptionCanceled) {
	reason := cancel.GetReason()
	if reason == "" {
		s.closeErrorHandler.OnClose()
		return
	}
	s.closeErrorHandler.Error(errors.New(reason))
}

func (s *DevicesSubscription) runRecv() {
	for {
		ev, err := s.client.Recv()
		if errors.Is(err, io.EOF) {
			s.cancelAndHandlerError(nil)
			s.closeErrorHandler.OnClose()
			return
		}
		if err != nil {
			s.cancelAndHandlerError(err)
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			s.handleCancel(cancel)
			return
		}

		if err := s.handleEvent(ev); err != nil {
			s.cancelAndHandlerError(err)
			return
		}
	}
}

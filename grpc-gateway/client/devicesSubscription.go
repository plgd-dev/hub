package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

// DeviceOnlineHandler handler of events.
type DeviceOnlineHandler = interface {
	HandleDeviceOnline(ctx context.Context, val *pb.Event_DeviceOnline) error
	SubscriptionHandler
}

// DeviceOfflineHandler handler of events.
type DeviceOfflineHandler = interface {
	HandleDeviceOffline(ctx context.Context, val *pb.Event_DeviceOffline) error
	SubscriptionHandler
}

// DeviceRegisteredHandler handler of events.
type DeviceRegisteredHandler = interface {
	HandleDeviceRegistered(ctx context.Context, val *pb.Event_DeviceRegistered) error
	SubscriptionHandler
}

// DeviceUnregisteredHandler handler of events.
type DeviceUnregisteredHandler = interface {
	HandleDeviceUnregistered(ctx context.Context, val *pb.Event_DeviceUnregistered) error
	SubscriptionHandler
}

// DevicesSubscription subscription.
type DevicesSubscription struct {
	client                    pb.GrpcGateway_SubscribeForEventsClient
	subscriptionID            string
	handle                    SubscriptionHandler
	deviceOnlineHandler       DeviceOnlineHandler
	deviceOfflineHandler      DeviceOfflineHandler
	deviceRegisteredHandler   DeviceRegisteredHandler
	deviceUnregisteredHandler DeviceUnregisteredHandler

	wait     func()
	canceled uint32
}

// NewDevicesSubscription creates new devices subscriptions to listen events: device online, device offline, device registered, device unregistered.
// JWT token must be stored in context for grpc call.
func (c *Client) NewDevicesSubscription(ctx context.Context, handle SubscriptionHandler) (*DevicesSubscription, error) {
	var deviceOnlineHandler DeviceOnlineHandler
	var deviceOfflineHandler DeviceOfflineHandler
	var deviceRegisteredHandler DeviceRegisteredHandler
	var deviceUnregisteredHandler DeviceUnregisteredHandler
	filterEvents := make([]pb.SubscribeForEvents_DevicesEventFilter_Event, 0, 1)
	if v, ok := handle.(DeviceOnlineHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DevicesEventFilter_ONLINE)
		deviceOnlineHandler = v
	}
	if v, ok := handle.(DeviceOfflineHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DevicesEventFilter_OFFLINE)
		deviceOfflineHandler = v
	}
	if v, ok := handle.(DeviceRegisteredHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DevicesEventFilter_REGISTERED)
		deviceRegisteredHandler = v
	}
	if v, ok := handle.(DeviceUnregisteredHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DevicesEventFilter_UNREGISTERED)
		deviceUnregisteredHandler = v
	}

	if deviceOnlineHandler == nil && deviceOfflineHandler == nil && deviceRegisteredHandler == nil && deviceUnregisteredHandler == nil {
		return nil, fmt.Errorf("invalid handler - it's supports: ResourceContentChangedHandler")
	}
	client, err := c.gateway.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_DevicesEvent{
			DevicesEvent: &pb.SubscribeForEvents_DevicesEventFilter{
				FilterEvents: filterEvents,
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
		return nil, fmt.Errorf(op.GetErrorStatus().GetMessage())
	}

	var wg sync.WaitGroup
	sub := &DevicesSubscription{
		client:                    client,
		handle:                    handle,
		subscriptionID:            ev.GetSubscriptionId(),
		deviceOnlineHandler:       deviceOnlineHandler,
		deviceOfflineHandler:      deviceOfflineHandler,
		deviceRegisteredHandler:   deviceRegisteredHandler,
		deviceUnregisteredHandler: deviceUnregisteredHandler,
		wait:                      wg.Wait,
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
	if !atomic.CompareAndSwapUint32(&s.canceled, s.canceled, 1) {
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

func (s *DevicesSubscription) runRecv() {
	for {
		ev, err := s.client.Recv()
		if err == io.EOF {
			s.handle.OnClose()
			return
		}
		if err != nil {
			s.handle.Error(err)
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			reason := cancel.GetReason()
			if reason == "" {
				s.handle.OnClose()
			}
			s.handle.Error(fmt.Errorf(reason))
			return
		}

		if ct := ev.GetDeviceOnline(); ct != nil {
			err = s.deviceOnlineHandler.HandleDeviceOnline(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else if ct := ev.GetDeviceOffline(); ct != nil {
			err = s.deviceOfflineHandler.HandleDeviceOffline(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else if ct := ev.GetDeviceRegistered(); ct != nil {
			err = s.deviceRegisteredHandler.HandleDeviceRegistered(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else if ct := ev.GetDeviceUnregistered(); ct != nil {
			err = s.deviceUnregisteredHandler.HandleDeviceUnregistered(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else {
			s.Cancel()
			s.handle.Error(fmt.Errorf("unknown event occurs on recv resource content changed: %+v", ev))
			return
		}
	}
}

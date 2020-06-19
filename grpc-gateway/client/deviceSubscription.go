package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

// ResourcePublishedHandler handler of events.
type ResourcePublishedHandler = interface {
	HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error
	SubscriptionHandler
}

// ResourceUnpublishedHandler handler of events.
type ResourceUnpublishedHandler = interface {
	HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error
	SubscriptionHandler
}

// ResourceUpdatePendingHandler handler of events
type ResourceUpdatePendingHandler = interface {
	HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error
	SubscriptionHandler
}

// ResourceUpdatedHandler handler of events
type ResourceUpdatedHandler = interface {
	HandleResourceUpdated(ctx context.Context, val *pb.Event_ResourceUpdated) error
	SubscriptionHandler
}

// ResourceRetrievePendingHandler handler of events
type ResourceRetrievePendingHandler = interface {
	HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error
	SubscriptionHandler
}

// ResourceRetrievedHandler handler of events
type ResourceRetrievedHandler = interface {
	HandleResourceRetrieved(ctx context.Context, val *pb.Event_ResourceRetrieved) error
	SubscriptionHandler
}

// DeviceSubscription subscription.
type DeviceSubscription struct {
	client                         pb.GrpcGateway_SubscribeForEventsClient
	subscriptionID                 string
	resourcePublishedHandler       ResourcePublishedHandler
	resourceUnpublishedHandler     ResourceUnpublishedHandler
	resourceUpdatePendingHandler   ResourceUpdatePendingHandler
	resourceUpdatedHandler         ResourceUpdatedHandler
	resourceRetrievePendingHandler ResourceRetrievePendingHandler
	resourceRetrievedHandler       ResourceRetrievedHandler
	closeErrorHandler              SubscriptionHandler

	wait     func()
	canceled uint32
}

// NewDeviceSubscription creates new devices subscriptions to listen events: resource published, resource unpublished.
// JWT token must be stored in context for grpc call.
func (c *Client) NewDeviceSubscription(ctx context.Context, deviceID string, handle SubscriptionHandler) (*DeviceSubscription, error) {
	return NewDeviceSubscription(ctx, deviceID, handle, handle, c.gateway)
}

// NewDeviceSubscription creates new devices subscriptions to listen events: resource published, resource unpublished.
// JWT token must be stored in context for grpc call.
func NewDeviceSubscription(ctx context.Context, deviceID string, closeErrorHandler SubscriptionHandler, handle SubscriptionHandler, gwClient pb.GrpcGatewayClient) (*DeviceSubscription, error) {
	var resourcePublishedHandler ResourcePublishedHandler
	var resourceUnpublishedHandler ResourceUnpublishedHandler
	var resourceUpdatePendingHandler ResourceUpdatePendingHandler
	var resourceUpdatedHandler ResourceUpdatedHandler
	var resourceRetrievePendingHandler ResourceRetrievePendingHandler
	var resourceRetrievedHandler ResourceRetrievedHandler
	filterEvents := make([]pb.SubscribeForEvents_DeviceEventFilter_Event, 0, 1)
	if v, ok := handle.(ResourcePublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED)
		resourcePublishedHandler = v
	}
	if v, ok := handle.(ResourceUnpublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED)
		resourceUnpublishedHandler = v
	}
	if v, ok := handle.(ResourceUpdatePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING)
		resourceUpdatePendingHandler = v
	}
	if v, ok := handle.(ResourceUpdatedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED)
		resourceUpdatedHandler = v
	}
	if v, ok := handle.(ResourceRetrievePendingHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING)
		resourceRetrievePendingHandler = v
	}
	if v, ok := handle.(ResourceRetrievedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED)
		resourceRetrievedHandler = v
	}

	if resourcePublishedHandler == nil &&
		resourceUnpublishedHandler == nil &&
		resourceUpdatePendingHandler == nil &&
		resourceUpdatedHandler == nil &&
		resourceRetrievePendingHandler == nil &&
		resourceRetrievedHandler == nil {
		return nil, fmt.Errorf("invalid handler - it's supports: ResourcePublishedHandler, ResourceUnpublishedHandler, ResourceUpdatePendingHandler, ResourceUpdatedHandler, ResourceRetrievePendingHandler, ResourceRetrievedHandler")
	}
	client, err := gwClient.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_DeviceEvent{
			DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
				DeviceId:     deviceID,
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
	sub := &DeviceSubscription{
		client:                         client,
		closeErrorHandler:              closeErrorHandler,
		subscriptionID:                 ev.GetSubscriptionId(),
		resourcePublishedHandler:       resourcePublishedHandler,
		resourceUnpublishedHandler:     resourceUnpublishedHandler,
		resourceUpdatePendingHandler:   resourceUpdatePendingHandler,
		resourceUpdatedHandler:         resourceUpdatedHandler,
		resourceRetrievePendingHandler: resourceRetrievePendingHandler,
		resourceRetrievedHandler:       resourceRetrievedHandler,
		wait:                           wg.Wait,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sub.runRecv()
	}()

	return sub, nil
}

// Cancel cancels subscription.
func (s *DeviceSubscription) Cancel() (wait func(), err error) {
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
func (s *DeviceSubscription) ID() string {
	return s.subscriptionID
}

func (s *DeviceSubscription) runRecv() {
	for {
		ev, err := s.client.Recv()
		if err == io.EOF {
			s.Cancel()
			s.closeErrorHandler.OnClose()
			return
		}
		if err != nil {
			s.Cancel()
			s.closeErrorHandler.Error(err)
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			s.Cancel()
			reason := cancel.GetReason()
			if reason == "" {
				s.closeErrorHandler.OnClose()
				return
			}
			s.closeErrorHandler.Error(fmt.Errorf(reason))
			return
		}

		if ct := ev.GetResourcePublished(); ct != nil {
			err = s.resourcePublishedHandler.HandleResourcePublished(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else if ct := ev.GetResourceUnpublished(); ct != nil {
			err = s.resourceUnpublishedHandler.HandleResourceUnpublished(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else if ct := ev.GetResourceUpdatePending(); ct != nil {
			err = s.resourceUpdatePendingHandler.HandleResourceUpdatePending(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else if ct := ev.GetResourceUpdated(); ct != nil {
			err = s.resourceUpdatedHandler.HandleResourceUpdated(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else if ct := ev.GetResourceRetrievePending(); ct != nil {
			err = s.resourceRetrievePendingHandler.HandleResourceRetrievePending(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else if ct := ev.GetResourceRetrieved(); ct != nil {
			err = s.resourceRetrievedHandler.HandleResourceRetrieved(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.closeErrorHandler.Error(err)
				return
			}
		} else {
			s.Cancel()
			s.closeErrorHandler.Error(fmt.Errorf("unknown event %T occurs on recv resource: %+v", ev, ev))
			return
		}
	}
}

func ToDeviceSubscription(v interface{}, ok bool) *DeviceSubscription {
	if !ok {
		return nil
	}
	if v == nil {
		return nil
	}
	s, ok := v.(*DeviceSubscription)
	if ok {
		return s
	}
	return nil
}

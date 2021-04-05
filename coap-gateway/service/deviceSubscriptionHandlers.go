package service

import (
	"context"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/log"
)

type deviceSubscriptionHandlers struct {
	client   *Client
	deviceID string
	mutex    sync.Mutex
}

func NewDeviceSubscriptionHandlers(client *Client, deviceID string) *deviceSubscriptionHandlers {
	return &deviceSubscriptionHandlers{
		client:   client,
		deviceID: deviceID,
	}
}

func (h *deviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		err := h.client.updateResource(ctx, val)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

func (h *deviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		err := h.client.retrieveResource(ctx, val)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

func (h *deviceSubscriptionHandlers) HandleResourceDeletePending(ctx context.Context, val *pb.Event_ResourceDeletePending) error {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		err := h.client.deleteResource(ctx, val)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

func (h *deviceSubscriptionHandlers) HandleResourceCreatePending(ctx context.Context, val *pb.Event_ResourceCreatePending) error {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		err := h.client.createResource(ctx, val)
		if err != nil {
			log.Error(err)
		}
	})
	return nil
}

func (h *deviceSubscriptionHandlers) Error(err error) {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending, ResourceDeletePending) ends with error: %v", h.deviceID, err)
		h.client.Close()
	})
}

func (h *deviceSubscriptionHandlers) OnClose() {
	h.client.server.taskQueue.Submit(func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		log.Debugf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending, ResourceDeletePending) was closed", h.deviceID)
		cancelSubscription := h.client.unsetCancelDeviceSubscription()
		if cancelSubscription != nil {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			cancelSubscription(ctx)
		}
	})
}

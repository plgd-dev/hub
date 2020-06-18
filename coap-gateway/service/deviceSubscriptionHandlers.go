package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
)

type deviceSubscriptionHandlers struct {
	*Client
	deviceID string
}

func (client *deviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
	return client.updateResource(ctx, val)
}

func (client *deviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
	return client.retrieveResource(ctx, val)
}

func (client *deviceSubscriptionHandlers) Error(err error) {
	log.Error(err)
	client.Close()
}

func (client *deviceSubscriptionHandlers) OnClose() {
	log.Debugf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", client.deviceID)
}

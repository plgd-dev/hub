package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

type deviceSubscriptionHandlers struct {
	onResourceUpdatePending   func(ctx context.Context, val *pb.Event_ResourceUpdatePending) error
	onResourceRetrievePending func(ctx context.Context, val *pb.Event_ResourceRetrievePending) error
	onClose                   func()
	onError                   func(err error)
}

func (h *deviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
	return h.onResourceUpdatePending(ctx, val)
}

func (h *deviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
	return h.onResourceRetrievePending(ctx, val)
}

func (h *deviceSubscriptionHandlers) Error(err error) {
	h.onError(err)
}

func (h *deviceSubscriptionHandlers) OnClose() {
	h.onClose()
}

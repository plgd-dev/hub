package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

type resourceSubscriptionHandlers struct {
	onChange func(ctx context.Context, val *pb.Event_ResourceChanged) error
	onClose  func()
	onError  func(err error)
}

func (h *resourceSubscriptionHandlers) HandleResourceContentChanged(ctx context.Context, val *pb.Event_ResourceChanged) error {
	return h.onChange(ctx, val)
}

func (h *resourceSubscriptionHandlers) Error(err error) {
	h.onError(err)
}

func (h *resourceSubscriptionHandlers) OnClose() {
	h.onClose()
}

package service

import (
	"context"

	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type resourceSubscriptionHandlers struct {
	onChange func(ctx context.Context, val *events.ResourceChanged) error
	onClose  func()
	onError  func(err error)
}

func (h *resourceSubscriptionHandlers) HandleResourceContentChanged(ctx context.Context, val *events.ResourceChanged) error {
	return h.onChange(ctx, val)
}

func (h *resourceSubscriptionHandlers) Error(err error) {
	h.onError(err)
}

func (h *resourceSubscriptionHandlers) OnClose() {
	h.onClose()
}

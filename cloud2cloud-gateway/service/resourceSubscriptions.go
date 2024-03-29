package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type resourceSubscriptionHandler struct {
	subData   *SubscriptionData
	emitEvent emitEventFunc
}

func (h *resourceSubscriptionHandler) HandleResourceContentChanged(ctx context.Context, val *raEvents.ResourceChanged) error {
	if val.GetStatus() != commands.Status_OK && val.GetStatus() != commands.Status_UNKNOWN {
		return fmt.Errorf("resourceSubscriptionHandler.HandleResourceContentChanged: cannot emit event for bad status %v of response", val.GetStatus())
	}
	var rep interface{}
	err := unmarshalContent(val.GetContent(), &rep)
	if err != nil {
		return fmt.Errorf("resourceSubscriptionHandler.HandleResourceContentChanged: cannot emit event: cannot unmarshal content: %w", err)
	}
	remove, err := h.emitEvent(ctx, events.EventType_ResourceChanged, h.subData.Data(), h.subData.IncrementSequenceNumber, rep)
	if err != nil {
		log.Errorf("resourceSubscriptionHandler.HandleResourceContentChanged: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

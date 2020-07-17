package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
)

type resourceSubsciptionHandler struct {
	subData   *SubscriptionData
	emitEvent emitEventFunc
}

func (h *resourceSubsciptionHandler) HandleResourceContentChanged(ctx context.Context, val *pb.Event_ResourceChanged) error {
	if val.GetStatus() != pb.Status_OK && val.GetStatus() != pb.Status_UNKNOWN {
		return fmt.Errorf("resourceSubsciptionHandler.HandleResourceContentChanged: cannot emit event for bad status %v of response", val.GetStatus())
	}
	rep, err := unmarshalContent(val.GetContent())
	if err != nil {
		return fmt.Errorf("resourceSubsciptionHandler.HandleResourceContentChanged: cannot emit event: cannot unmarshal content: %w", err)
	}
	remove, err := h.emitEvent(ctx, events.EventType_ResourceChanged, h.subData.Data(), h.subData.IncrementSequenceNumber, rep)
	if err != nil {
		log.Errorf("resourceSubsciptionHandler.HandleResourceContentChanged: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

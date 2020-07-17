package service

import (
	"context"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

type deviceSubsciptionHandler struct {
	subData *SubscriptionData
}

func fixResourceLink(r schema.ResourceLink) schema.ResourceLink {
	r.Href = getHref(r.DeviceID, r.Href)
	r.ID = ""
	return r
}

func (h *deviceSubsciptionHandler) HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error {
	toSend := make([]schema.ResourceLink, 0, 32)
	for _, l := range val.GetLinks() {
		if l.GetHref() == cloud.StatusHref {
			continue
		}
		toSend = append(toSend, fixResourceLink(l.ToSchema()))
	}
	if len(toSend) == 0 {
		return nil
	}
	remove, err := emitEvent(ctx, events.EventType_ResourcesPublished, h.subData.Data(), h.subData.IncrementSequenceNumber, toSend)
	if err != nil {
		log.Errorf("deviceSubsciptionHandler.HandleResourcePublished: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *deviceSubsciptionHandler) HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error {
	toSend := make([]schema.ResourceLink, 0, 32)
	for _, l := range val.GetLinks() {
		if l.GetHref() == cloud.StatusHref {
			continue
		}
		toSend = append(toSend, fixResourceLink(l.ToSchema()))
	}
	if len(toSend) == 0 {
		return nil
	}
	remove, err := emitEvent(ctx, events.EventType_ResourcesUnpublished, h.subData.Data(), h.subData.IncrementSequenceNumber, toSend)
	if err != nil {
		log.Errorf("deviceSubsciptionHandler.HandleResourceUnpublished: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

type resourcePublishedHandler struct {
	h *deviceSubsciptionHandler
}

func (h *resourcePublishedHandler) HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error {
	return h.h.HandleResourcePublished(ctx, val)
}

type resourceUnpublishedHandler struct {
	h *deviceSubsciptionHandler
}

func (h *resourceUnpublishedHandler) HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error {
	return h.h.HandleResourceUnpublished(ctx, val)
}

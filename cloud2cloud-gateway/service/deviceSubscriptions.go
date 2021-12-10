package service

import (
	"context"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	raEvents "github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/kit/v2/log"
)

type deviceSubscriptionHandler struct {
	subData   *SubscriptionData
	emitEvent emitEventFunc
}

// TODO:
// - switch to schema.ResourceLink
// - currently CTT5.3.2 test cases expects that every resource link in ResourceUnpublished notification
// has href, rt, if fields (even if empty)
// - this should be clarification in the OCF specification and then either the test case or
// schema.ResourceLink should be modified to follow the specification
type resourceLink struct {
	DeviceID      string   `json:"di,omitempty"`
	Href          string   `json:"href"`
	ResourceTypes []string `json:"rt"`
	Interfaces    []string `json:"if"`
}

func fixResourceLink(r schema.ResourceLink) schema.ResourceLink {
	r.Href = getHref(r.DeviceID, r.Href)
	r.ID = ""
	return r
}

func (h *deviceSubscriptionHandler) HandleResourcePublished(ctx context.Context, val *raEvents.ResourceLinksPublished) error {
	toSend := make([]schema.ResourceLink, 0, len(val.GetResources()))
	for _, l := range val.GetResources() {
		toSend = append(toSend, fixResourceLink(l.ToSchema()))
	}
	if len(toSend) == 0 && len(val.GetResources()) > 0 {
		return nil
	}
	remove, err := h.emitEvent(ctx, events.EventType_ResourcesPublished, h.subData.Data(), h.subData.IncrementSequenceNumber, toSend)
	if err != nil {
		log.Errorf("deviceSubscriptionHandler.HandleResourcePublished: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

func (h *deviceSubscriptionHandler) HandleResourceUnpublished(ctx context.Context, val *raEvents.ResourceLinksUnpublished) error {
	toSend := make([]resourceLink, 0, len(val.GetHrefs()))
	for _, l := range val.GetHrefs() {
		toSend = append(toSend, resourceLink{
			DeviceID: val.GetDeviceId(),
			Href:     getHref(val.GetDeviceId(), l),
		})
	}
	if len(toSend) == 0 && len(val.GetHrefs()) > 0 {
		return nil
	}
	remove, err := h.emitEvent(ctx, events.EventType_ResourcesUnpublished, h.subData.Data(), h.subData.IncrementSequenceNumber, toSend)
	if err != nil {
		log.Errorf("deviceSubscriptionHandler.HandleResourceUnpublished: cannot emit event: %v", err)
	}
	if remove {
		return err
	}
	return nil
}

type resourcePublishedHandler struct {
	h *deviceSubscriptionHandler
}

func (h *resourcePublishedHandler) HandleResourcePublished(ctx context.Context, val *raEvents.ResourceLinksPublished) error {
	return h.h.HandleResourcePublished(ctx, val)
}

type resourceUnpublishedHandler struct {
	h *deviceSubscriptionHandler
}

func (h *resourceUnpublishedHandler) HandleResourceUnpublished(ctx context.Context, val *raEvents.ResourceLinksUnpublished) error {
	return h.h.HandleResourceUnpublished(ctx, val)
}

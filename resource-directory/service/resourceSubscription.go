package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/strings"
)

type resourceSubscription struct {
	*subscription
	resourceEvent *pb.SubscribeForEvents_ResourceEventFilter
}

func NewResourceSubscription(id, userID, token string, send SendEventFunc, resourceProjection *Projection, resourceEvent *pb.SubscribeForEvents_ResourceEventFilter) *resourceSubscription {
	log.Debugf("subscription.NewResourceSubscription %v %+v", id, resourceEvent.GetResourceId())
	defer log.Debugf("subscription.NewResourceSubscription %v done", id)
	return &resourceSubscription{
		subscription:  NewSubscription(userID, id, token, send, resourceProjection),
		resourceEvent: resourceEvent,
	}
}

func (s *resourceSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	log.Debugf("subscriptions.SubscribeForResourceEvent.resourceProjection.Register")
	if !currentDevices[s.ResourceID().GetDeviceId()] {
		return fmt.Errorf("device %v not found", s.ResourceID().GetDeviceId())
	}

	created, err := s.RegisterToProjection(ctx, s.ResourceID().GetDeviceId())
	if err != nil {
		return fmt.Errorf("cannot register to resource projection: %w", err)
	}
	log.Debugf("subscriptions.SubscribeForResourceEvent.resourceProjection.Register, created=%v", created)

	deviceIDFilter := make(strings.Set)
	deviceIDFilter.Add(s.ResourceID().GetDeviceId())
	resLinks, err := s.resourceProjection.GetResourceLinks(ctx, deviceIDFilter, nil)
	if err != nil {
		return fmt.Errorf("links not available for device %v", s.ResourceID().GetDeviceId())
	}

	if _, present := resLinks[s.ResourceID().GetDeviceId()][s.ResourceID().GetHref()]; !present {
		return fmt.Errorf("not found")
	}

	models := s.resourceProjection.Models(s.ResourceID())
	if len(models) == 0 {
		err = s.resourceProjection.ForceUpdate(ctx, s.ResourceID())
		if err != nil {
			return fmt.Errorf("cannot load resources for device: %w", err)
		}
		models = s.resourceProjection.Models(s.ResourceID())
	}

	if len(models) == 0 {
		return nil
	}
	res := models[0].(*resourceProjection).Clone()
	if res.content == nil {
		return nil
	}

	for _, f := range s.resourceEvent.FilterEvents {
		switch f {
		case pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED:
			if res.content.GetStatus() == commands.Status_UNKNOWN {
				continue
			}
			err := res.onResourceChangedLocked(ctx, s.NotifyOfContentChangedResource)
			if err != nil {
				return fmt.Errorf("cannot send resource content changed: %w", err)
			}
		}
	}
	return nil
}

func (s *resourceSubscription) ResourceID() *commands.ResourceId {
	return s.resourceEvent.GetResourceId()
}

func (s *resourceSubscription) NotifyOfContentChangedResource(ctx context.Context, resourceChanged pb.Event_ResourceChanged, version uint64) error {
	deviceID := resourceChanged.GetResourceId().GetDeviceId()
	href := resourceChanged.GetResourceId().GetHref()
	if s.FilterByVersion(deviceID, href, "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.resourceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceChanged_{
			ResourceChanged: &resourceChanged,
		},
	})
}

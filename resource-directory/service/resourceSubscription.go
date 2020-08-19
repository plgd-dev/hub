package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	cqrsRA "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
)

type resourceSubscription struct {
	*subscription
	resourceEvent *pb.SubscribeForEvents_ResourceEventFilter
}

func NewResourceSubscription(id, userID string, send SendEventFunc, resourceProjection *Projection, resourceEvent *pb.SubscribeForEvents_ResourceEventFilter) *resourceSubscription {
	log.Debugf("subscription.NewResourceSubscription %v", id)
	defer log.Debugf("subscription.NewResourceSubscription %v done", id)
	return &resourceSubscription{
		subscription:  NewSubscription(userID, id, send, resourceProjection),
		resourceEvent: resourceEvent,
	}
}

func (s *resourceSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	log.Debugf("subscriptions.SubscribeForResourceEvent.resourceProjection.Register")
	if !currentDevices[s.DeviceID()] {
		return fmt.Errorf("device %v not found", s.DeviceID())
	}

	created, err := s.RegisterToProjection(ctx, s.DeviceID())
	if err != nil {
		return fmt.Errorf("cannot register to resource projection: %w", err)
	}
	log.Debugf("subscriptions.SubscribeForResourceEvent.resourceProjection.Register, created=%v", created)

	resourceID := cqrsRA.MakeResourceId(s.DeviceID(), s.Href())
	models := s.resourceProjection.Models(s.DeviceID(), resourceID)
	if len(models) == 0 {
		err = s.resourceProjection.ForceUpdate(ctx, s.DeviceID(), resourceID)
		if err != nil {
			return fmt.Errorf("cannot load resources for device: %w", err)
		}
		models = s.resourceProjection.Models(s.DeviceID(), resourceID)
	}

	if len(models) == 0 {
		return fmt.Errorf("cannot load resource models %v%v: %w", s.DeviceID(), s.Href(), err)
	}
	res := models[0].(*resourceCtx).Clone()
	if res.content == nil {
		return nil
	}

	for _, f := range s.resourceEvent.FilterEvents {
		switch f {
		case pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED:
			if res.content.GetStatus() == pbRA.Status_UNKNOWN {
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

func (s *resourceSubscription) DeviceID() string {
	return s.resourceEvent.GetResourceId().GetDeviceId()
}

func (s *resourceSubscription) Href() string {
	return s.resourceEvent.GetResourceId().GetHref()
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
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceChanged_{
			ResourceChanged: &resourceChanged,
		},
	})
}

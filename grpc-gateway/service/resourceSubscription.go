package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/ocf-cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	cqrsRA "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs"
	projectionRA "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/projection"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
)

type resourceSubscription struct {
	*subscription
	resourceEvent *pb.SubscribeForEvents_ResourceEventFilter
}

func NewResourceSubscription(id, userID string, send SendEventFunc, resourceProjection *projectionRA.Projection, resourceEvent *pb.SubscribeForEvents_ResourceEventFilter) *resourceSubscription {
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
			if res.content.GetStatus() != pbRA.Status_OK {
				return fmt.Errorf("unable to subscribe to resource %v%v: device response: %v", res.resource.GetDeviceId(), res.resource.GetHref(), res.content.GetStatus())
			}
			content := makeContent(res.content.GetContent())
			err := s.NotifyOfContentChangedResource(ctx, content, res.onResourceChangedVersion)
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
	return s.resourceEvent.GetResourceId().GetResourceLinkHref()
}

func (s *resourceSubscription) NotifyOfContentChangedResource(ctx context.Context, content pb.Content, version uint64) error {
	deviceID := s.resourceEvent.GetResourceId().GetDeviceId()
	href := s.resourceEvent.GetResourceId().GetResourceLinkHref()
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
		Type: &pb.Event_ResourceContentChanged{
			ResourceContentChanged: &pb.Event_ResourceChanged{
				ResourceId: &pb.ResourceId{
					DeviceId:         deviceID,
					ResourceLinkHref: href,
				},
				Content: &content,
			},
		},
	})
}

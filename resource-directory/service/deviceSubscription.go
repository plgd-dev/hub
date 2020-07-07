package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
)

type deviceSubscription struct {
	*subscription
	deviceEvent *pb.SubscribeForEvents_DeviceEventFilter
}

func NewDeviceSubscription(id, userID string, send SendEventFunc, resourceProjection *Projection, deviceEvent *pb.SubscribeForEvents_DeviceEventFilter) *deviceSubscription {
	log.Debugf("subscription.NewDeviceSubscription %v", id)
	defer log.Debugf("subscription.NewDeviceSubscription %v done", id)
	return &deviceSubscription{
		subscription: NewSubscription(userID, id, send, resourceProjection),
		deviceEvent:  deviceEvent,
	}
}

func (s *deviceSubscription) DeviceID() string {
	return s.deviceEvent.GetDeviceId()
}

func (s *deviceSubscription) NotifyOfPublishedResource(ctx context.Context, link pb.ResourceLink, version uint64) error {
	if s.FilterByVersion(link.GetDeviceId(), link.GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourcePublished_{
			ResourcePublished: &pb.Event_ResourcePublished{
				Link: &link,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUnpublishedResource(ctx context.Context, link pb.ResourceLink, version uint64) error {
	if s.FilterByVersion(link.GetDeviceId(), link.GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUnpublished_{
			ResourceUnpublished: &pb.Event_ResourceUnpublished{
				Link: &link,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatePendingResource(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error {
	if s.FilterByVersion(updatePending.GetResourceId().GetDeviceId(), updatePending.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdatePending_{
			ResourceUpdatePending: &updatePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatedResource(ctx context.Context, updated pb.Event_ResourceUpdated, version uint64) error {
	if s.FilterByVersion(updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdated_{
			ResourceUpdated: &updated,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievePendingResource(ctx context.Context, retrievePending pb.Event_ResourceRetrievePending, version uint64) error {
	if s.FilterByVersion(retrievePending.GetResourceId().GetDeviceId(), retrievePending.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrievePending_{
			ResourceRetrievePending: &retrievePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievedResource(ctx context.Context, retrieved pb.Event_ResourceRetrieved, version uint64) error {
	if s.FilterByVersion(retrieved.GetResourceId().GetDeviceId(), retrieved.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED {
			found = true
		}
	}
	if !found {
		return nil
	}
	return s.Send(ctx, pb.Event{
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrieved_{
			ResourceRetrieved: &retrieved,
		},
	})
}

func (s *deviceSubscription) initSendResourcesPublished(ctx context.Context) error {
	models := s.resourceProjection.Models(s.DeviceID(), "")
	for _, model := range models {
		link, version, ok := makeLinkRepresentation(pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED, model)
		if !ok {
			continue
		}
		err := s.NotifyOfPublishedResource(ctx, link, version)
		if err != nil {
			return fmt.Errorf("cannot send resource published: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesUnpublished(ctx context.Context) error {
	models := s.resourceProjection.Models(s.DeviceID(), "")
	for _, model := range models {
		link, version, ok := makeLinkRepresentation(pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED, model)
		if !ok {
			continue
		}
		err := s.NotifyOfUnpublishedResource(ctx, link, version)
		if err != nil {
			return fmt.Errorf("cannot send resource published: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesUpdatePending(ctx context.Context) error {
	models := s.resourceProjection.Models(s.DeviceID(), "")
	for _, model := range models {
		c := model.(*resourceCtx).Clone()
		err := c.onResourceUpdatePendingLocked(ctx, s.NotifyOfUpdatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesRetrievePending(ctx context.Context) error {
	models := s.resourceProjection.Models(s.DeviceID(), "")
	for _, model := range models {
		c := model.(*resourceCtx).Clone()
		err := c.onResourceRetrievePendingLocked(ctx, s.NotifyOfRetrievePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) Init(ctx context.Context, currentDevices map[string]bool) error {
	if !currentDevices[s.DeviceID()] {
		return fmt.Errorf("device %v not found", s.DeviceID())
	}
	_, err := s.RegisterToProjection(ctx, s.DeviceID())
	if err != nil {
		return fmt.Errorf("cannot register to resource projection: %w", err)
	}

	for _, f := range s.deviceEvent.GetFilterEvents() {
		switch f {
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED:
			err = s.initSendResourcesPublished(ctx)
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED:
			err = s.initSendResourcesUnpublished(ctx)
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING:
			err = s.initSendResourcesUpdatePending(ctx)
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING:
			err = s.initSendResourcesRetrievePending(ctx)
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED:
			// do nothing
		}
		if err != nil {
			return err
		}
	}
	return nil
}

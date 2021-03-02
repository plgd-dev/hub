package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/log"
	"go.uber.org/atomic"
)

type deviceSubscription struct {
	*subscription
	deviceEvent                  *pb.SubscribeForEvents_DeviceEventFilter
	isInitializedResourcePublish atomic.Bool
}

func NewDeviceSubscription(id, userID, token string, send SendEventFunc, resourceProjection *Projection, deviceEvent *pb.SubscribeForEvents_DeviceEventFilter) *deviceSubscription {
	log.Debugf("subscription.NewDeviceSubscription %v", id)
	defer log.Debugf("subscription.NewDeviceSubscription %v done", id)
	return &deviceSubscription{
		subscription: NewSubscription(userID, id, token, send, resourceProjection),
		deviceEvent:  deviceEvent,
	}
}

func (s *deviceSubscription) DeviceID() string {
	return s.deviceEvent.GetDeviceId()
}

type ResourceLinks struct {
	links   []*pb.ResourceLink
	version uint64
	isInit  bool
}

func (s *deviceSubscription) NotifyOfPublishedResourceLinks(ctx context.Context, links ResourceLinks) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if links.isInit {
		s.isInitializedResourcePublish.Store(true)
	}
	if !s.isInitializedResourcePublish.Load() {
		return nil
	}
	if len(links.links) == 0 || s.FilterByVersion(links.links[0].GetDeviceId(), commands.ResourceLinksHref, "res", links.version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourcePublished_{
			ResourcePublished: &pb.Event_ResourcePublished{
				Links: links.links,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUnpublishedResourceLinks(ctx context.Context, links ResourceLinks) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if len(links.links) == 0 || s.FilterByVersion(links.links[0].GetDeviceId(), commands.ResourceLinksHref, "res", links.version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUnpublished_{
			ResourceUnpublished: &pb.Event_ResourceUnpublished{
				Links: links.links,
			},
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatePendingResource(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(updatePending.GetResourceId().GetDeviceId(), updatePending.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdatePending_{
			ResourceUpdatePending: &updatePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfUpdatedResource(ctx context.Context, updated pb.Event_ResourceUpdated, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(updated.GetResourceId().GetDeviceId(), updated.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdated_{
			ResourceUpdated: &updated,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievePendingResource(ctx context.Context, retrievePending pb.Event_ResourceRetrievePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(retrievePending.GetResourceId().GetDeviceId(), retrievePending.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrievePending_{
			ResourceRetrievePending: &retrievePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfRetrievedResource(ctx context.Context, retrieved pb.Event_ResourceRetrieved, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(retrieved.GetResourceId().GetDeviceId(), retrieved.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrieved_{
			ResourceRetrieved: &retrieved,
		},
	})
}

func (s *deviceSubscription) NotifyOfDeletePendingResource(ctx context.Context, deletePending pb.Event_ResourceDeletePending, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_DELETE_PENDING {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(deletePending.GetResourceId().GetDeviceId(), deletePending.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeletePending_{
			ResourceDeletePending: &deletePending,
		},
	})
}

func (s *deviceSubscription) NotifyOfDeletedResource(ctx context.Context, deleted pb.Event_ResourceDeleted, version uint64) error {
	var found bool
	for _, f := range s.deviceEvent.GetFilterEvents() {
		if f == pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_DELETED {
			found = true
		}
	}
	if !found {
		return nil
	}
	if s.FilterByVersion(deleted.GetResourceId().GetDeviceId(), deleted.GetResourceId().GetHref(), "res", version) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeleted_{
			ResourceDeleted: &deleted,
		},
	})
}

func (s *deviceSubscription) initSendResourcesPublished(ctx context.Context) error {
	models := s.resourceProjection.Models(commands.NewResourceID(s.DeviceID(), commands.ResourceLinksHref))
	if len(models) != 1 {
		return nil
	}

	rlp, ok := models[0].(*resourceLinksProjection)
	if !ok {
		return fmt.Errorf("unexpected event type")
	}

	err := rlp.InitialNotifyOfPublishedResourceLinks(ctx, s)
	if err != nil {
		return fmt.Errorf("cannot send resource published: %w", err)
	}

	return nil
}

func (s *deviceSubscription) initSendResourcesUnpublished(ctx context.Context) error {
	err := s.NotifyOfUnpublishedResourceLinks(ctx, ResourceLinks{})
	if err != nil {
		return fmt.Errorf("cannot send resource published: %w", err)
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesUpdatePending(ctx context.Context) error {
	models, err := s.resourceProjection.GetResources(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, model := range models[s.DeviceID()] {
		err := model.Projection.onResourceUpdatePendingLocked(ctx, s.NotifyOfUpdatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesRetrievePending(ctx context.Context) error {
	models, err := s.resourceProjection.GetResources(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, model := range models[s.DeviceID()] {
		err := model.Projection.onResourceRetrievePendingLocked(ctx, s.NotifyOfRetrievePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource retrieve pending: %w", err)
		}
	}
	return nil
}

func (s *deviceSubscription) initSendResourcesDeletePending(ctx context.Context) error {
	models, err := s.resourceProjection.GetResources(ctx, []*commands.ResourceId{commands.NewResourceID(s.DeviceID(), "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, model := range models[s.DeviceID()] {
		err := model.Projection.onResourceDeletePendingLocked(ctx, s.NotifyOfDeletePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource delete pending: %w", err)
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
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_DELETE_PENDING:
			err = s.initSendResourcesDeletePending(ctx)
		case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UPDATED, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_RETRIEVED:
			// do nothing
		}
		if err != nil {
			return err
		}
	}
	return nil
}

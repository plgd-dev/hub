package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type ResourceLinksPublished struct {
	data   *events.ResourceLinksPublished
	isInit bool
}

func (s *subscription) NotifyOfPublishedResourceLinks(ctx context.Context, links ResourceLinksPublished) error {
	if s.filteredEvents&filterBitmaskResourcesPublished == 0 {
		return nil
	}
	if len(links.data.GetResources()) == 0 {
		return nil
	}
	if !s.initializeResource(commands.NewResourceID(links.data.GetDeviceId(), commands.ResourceLinksHref), links.isInit) {
		return nil
	}
	if !s.Filter(commands.NewResourceID(links.data.GetDeviceId(), commands.ResourceLinksHref), "res", links.data.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourcePublished{
			ResourcePublished: links.data,
		},
	})
}

func (s *subscription) NotifyOfUnpublishedResourceLinks(ctx context.Context, unpublishedResources *events.ResourceLinksUnpublished) error {
	if s.filteredEvents&filterBitmaskResourcesUnpublished == 0 {
		return nil
	}
	if len(unpublishedResources.GetHrefs()) == 0 {
		return nil
	}
	if !s.Filter(commands.NewResourceID(unpublishedResources.GetDeviceId(), commands.ResourceLinksHref), "res", unpublishedResources.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUnpublished{
			ResourceUnpublished: unpublishedResources,
		},
	})
}

func (s *subscription) initGetResourceLinkProjection(deviceID string) eventstore.Model {
	resourceID := commands.NewResourceID(deviceID, commands.ResourceLinksHref)
	value, _ := s.isInitializedResource.LoadOrStoreWithFunc(resourceID.ToUUID(), func(value interface{}) interface{} {
		v := value.(*isInitialized)
		v.Lock()
		return v
	}, func() interface{} {
		v := new(isInitialized)
		v.Lock()
		return v
	})
	v := value.(*isInitialized)
	defer v.Unlock()
	models := s.resourceProjection.Models(resourceID)
	if len(models) != 1 {
		v.initialized = true
		return nil
	}
	return models[0]
}

func (s *subscription) initSendResourcesPublished(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourcesPublished == 0 {
		return nil
	}
	model := s.initGetResourceLinkProjection(deviceID)
	if model == nil {
		return nil
	}

	rlp, ok := model.(*resourceLinksProjection)
	if !ok {
		return fmt.Errorf("unexpected event type")
	}

	err := rlp.InitialNotifyOfPublishedResourceLinks(ctx, s)
	if err != nil {
		return fmt.Errorf("cannot send resource published: %w", err)
	}

	return nil
}

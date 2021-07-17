package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

func (s *subscription) initSendResourcesUpdatePending(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourceUpdatePending == 0 {
		return nil
	}
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(deviceID, "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[deviceID] {
		err := resource.OnResourceUpdatePendingLocked(ctx, s.NotifyOfUpdatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *subscription) initSendResourcesRetrievePending(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourceRetrievePending == 0 {
		return nil
	}
	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(deviceID, "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[deviceID] {
		err := resource.OnResourceRetrievePendingLocked(ctx, s.NotifyOfRetrievePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource retrieve pending: %w", err)
		}
	}
	return nil
}

func (s *subscription) initSendResourcesDeletePending(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourceDeletePending == 0 {
		return nil
	}

	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(deviceID, "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[deviceID] {
		err := resource.OnResourceDeletePendingLocked(ctx, s.NotifyOfDeletePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource delete pending: %w", err)
		}
	}
	return nil
}

func (s *subscription) initSendResourcesCreatePending(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourceCreatePending == 0 {
		return nil
	}

	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(deviceID, "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource update pending: %w", err)
	}

	for _, resource := range resources[deviceID] {
		err := resource.OnResourceCreatePendingLocked(ctx, s.NotifyOfCreatePendingResource)
		if err != nil {
			return fmt.Errorf("cannot send resource update pending: %w", err)
		}
	}
	return nil
}

func (s *subscription) initSendResourcesChanged(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskResourceChanged == 0 {
		return nil
	}

	resources, err := s.resourceProjection.GetResourcesWithLinks(ctx, []*commands.ResourceId{commands.NewResourceID(deviceID, "")}, nil)
	if err != nil {
		return fmt.Errorf("cannot send resource changed: %w", err)
	}

	for _, resource := range resources[deviceID] {
		err := resource.OnResourceChangedLocked(ctx, s.NotifyOfResourceChanged)
		if err != nil {
			return fmt.Errorf("cannot send  resource changed: %w", err)
		}
	}
	return nil
}

func (s *subscription) NotifyOfUpdatePendingResource(ctx context.Context, updatePending *events.ResourceUpdatePending) error {
	if s.filteredEvents&filterBitmaskResourceUpdatePending == 0 {
		return nil
	}
	if !s.Filter(updatePending.GetResourceId(), "update", updatePending.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdatePending{
			ResourceUpdatePending: updatePending,
		},
	})
}

func (s *subscription) NotifyOfUpdatedResource(ctx context.Context, updated *events.ResourceUpdated) error {
	if s.filteredEvents&filterBitmaskResourceUpdated == 0 {
		return nil
	}
	if !s.Filter(updated.GetResourceId(), "update", updated.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceUpdated{
			ResourceUpdated: updated,
		},
	})
}

func (s *subscription) NotifyOfRetrievePendingResource(ctx context.Context, retrievePending *events.ResourceRetrievePending) error {
	if s.filteredEvents&filterBitmaskResourceRetrievePending == 0 {
		return nil
	}
	if !s.Filter(retrievePending.GetResourceId(), "retrieve", retrievePending.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrievePending{
			ResourceRetrievePending: retrievePending,
		},
	})
}

func (s *subscription) NotifyOfRetrievedResource(ctx context.Context, retrieved *events.ResourceRetrieved) error {
	if s.filteredEvents&filterBitmaskResourceRetrieved == 0 {
		return nil
	}
	if !s.Filter(retrieved.GetResourceId(), "retrieve", retrieved.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceRetrieved{
			ResourceRetrieved: retrieved,
		},
	})
}

func (s *subscription) NotifyOfDeletePendingResource(ctx context.Context, deletePending *events.ResourceDeletePending) error {
	if s.filteredEvents&filterBitmaskResourceDeletePending == 0 {
		return nil
	}
	if !s.Filter(deletePending.GetResourceId(), "delete", deletePending.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeletePending{
			ResourceDeletePending: deletePending,
		},
	})
}

func (s *subscription) NotifyOfDeletedResource(ctx context.Context, deleted *events.ResourceDeleted) error {
	if s.filteredEvents&filterBitmaskResourceDeleted == 0 {
		return nil
	}
	if !s.Filter(deleted.GetResourceId(), "delete", deleted.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceDeleted{
			ResourceDeleted: deleted,
		},
	})
}

func (s *subscription) NotifyOfCreatePendingResource(ctx context.Context, createPending *events.ResourceCreatePending) error {
	if s.filteredEvents&filterBitmaskResourceCreatePending == 0 {
		return nil
	}
	if !s.Filter(createPending.GetResourceId(), "create", createPending.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceCreatePending{
			ResourceCreatePending: createPending,
		},
	})
}

func (s *subscription) NotifyOfCreatedResource(ctx context.Context, created *events.ResourceCreated) error {
	if s.filteredEvents&filterBitmaskResourceCreated == 0 {
		return nil
	}
	if !s.Filter(created.GetResourceId(), "create", created.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceCreated{
			ResourceCreated: created,
		},
	})
}

func (s *subscription) NotifyOfResourceChanged(ctx context.Context, resourceChanged *events.ResourceChanged) error {
	if s.filteredEvents&filterBitmaskResourceChanged == 0 {
		return nil
	}
	if !s.Filter(resourceChanged.GetResourceId(), "res", resourceChanged.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		CorrelationId:  s.CorrelationID(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_ResourceChanged{
			ResourceChanged: resourceChanged,
		},
	})
}

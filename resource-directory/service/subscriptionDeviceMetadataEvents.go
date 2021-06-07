package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

func (s *subscription) initNotifyOfDevicesMetadata(ctx context.Context, deviceID string) error {
	if s.filteredEvents&filterBitmaskDeviceMetadataUpdated == 0 {
		return nil
	}
	statusResourceID := commands.NewResourceID(deviceID, commands.StatusHref)
	models := s.resourceProjection.Models(statusResourceID)
	if len(models) == 0 {
		return nil
	}
	res := models[0].(*deviceMetadataProjection)
	return res.InitialNotifyOfDeviceMetadata(ctx, s)
}

func (s *subscription) NotifyOfUpdatePendingDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	if s.filteredEvents&filterBitmaskDeviceMetadataUpdatePending == 0 {
		return nil
	}
	if !s.Filter(commands.NewResourceID(event.GetDeviceId(), commands.StatusHref), "res", event.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceMetadataUpdatePending{
			DeviceMetadataUpdatePending: event,
		},
	})
}

func (s *subscription) NotifyOfUpdatedDeviceMetadata(ctx context.Context, event *events.DeviceMetadataUpdated) error {
	if s.filteredEvents&filterBitmaskDeviceMetadataUpdated == 0 {
		return nil
	}
	if !s.Filter(commands.NewResourceID(event.GetDeviceId(), commands.StatusHref), "res", event.Version()) {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: event,
		},
	})
}

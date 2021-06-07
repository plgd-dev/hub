package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

func (s *subscription) NotifyOfRegisteredDevice(ctx context.Context, deviceIDs []string) error {
	for _, d := range deviceIDs {
		s.isInitialized.Store(d, true)
	}
	if s.filteredEvents&filterBitmaskDeviceRegistered == 0 {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceRegistered_{
			DeviceRegistered: &pb.Event_DeviceRegistered{
				DeviceIds: deviceIDs,
			},
		},
	})
}

func (s *subscription) NotifyOfUnregisteredDevice(ctx context.Context, deviceIDs []string) error {
	for _, d := range deviceIDs {
		s.isInitialized.Delete(d)
	}
	if s.filteredEvents&filterBitmaskDeviceUnregistered == 0 {
		return nil
	}
	return s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_DeviceUnregistered_{
			DeviceUnregistered: &pb.Event_DeviceUnregistered{
				DeviceIds: deviceIDs,
			},
		},
	})
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/identity-store/events"
	"github.com/plgd-dev/cloud/identity-store/pb"
	"github.com/plgd-dev/cloud/identity-store/persistence"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getUniqueDeviceIds(deviceIds []string) []string {
	devices := make(strings.Set)
	devices.Add(deviceIds...)
	delete(devices, "")
	return devices.ToSlice()
}

func getOwnerDevices(tx persistence.PersistenceTx, owner string) ([]string, error) {
	it := tx.RetrieveByOwner(owner)
	defer it.Close()
	if it.Err() != nil {
		return nil, fmt.Errorf("failed to obtain owned devices: %w", it.Err())
	}
	var deviceIds []string
	var d persistence.AuthorizedDevice
	for it.Next(&d) {
		deviceIds = append(deviceIds, d.DeviceID)
	}
	return deviceIds, nil
}

func (s *Service) publishDevicesUnregistered(ctx context.Context, owner, userID string, deviceIDs []string) error {
	v := events.Event{
		Type: &events.Event_DevicesUnregistered{
			DevicesUnregistered: &events.DevicesUnregistered{
				Owner:     owner,
				DeviceIds: deviceIDs,
				AuditContext: &events.AuditContext{
					UserId: userID,
				},
				Timestamp: pkgTime.UnixNano(time.Now()),
			},
		},
	}
	data, err := utils.Marshal(&v)
	if err != nil {
		return err
	}

	err = s.publisher.PublishData(ctx, events.GetDevicesUnregisteredSubject(owner), data)
	if err != nil {
		return err
	}

	err = s.publisher.Flush(ctx)
	if err != nil {
		return err
	}
	return nil
}

// DeleteDevices removes a devices from user.
func (s *Service) DeleteDevices(ctx context.Context, request *pb.DeleteDevicesRequest) (*pb.DeleteDevicesResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner, userID, err := s.parseTokenMD(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardFromError(codes.InvalidArgument, fmt.Errorf("cannot delete devices: %w", err)))
	}

	var deviceIds []string
	if len(request.DeviceIds) == 0 {
		var err error
		if deviceIds, err = getOwnerDevices(tx, owner); err != nil {
			return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete devices: %v", err))
		}
		if len(deviceIds) == 0 {
			return &pb.DeleteDevicesResponse{}, nil
		}
	} else {
		deviceIds = getUniqueDeviceIds(request.DeviceIds)
		if len(deviceIds) == 0 {
			return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete devices: invalid DeviceIds"))
		}
	}

	var deletedDeviceIds []string
	for _, deviceId := range deviceIds {
		_, ok, err := tx.Retrieve(deviceId, owner)
		if err != nil {
			return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot delete device('%v'): %v", deviceId, err.Error()))
		}
		if !ok {
			log.Debugf("cannot retrieve device('%v') by user('%v')", deviceId, owner)
			continue
		}

		err = tx.Delete(deviceId, owner)
		if err != nil {
			return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot delete device('%v'): not found", deviceId))
		}

		deletedDeviceIds = append(deletedDeviceIds, deviceId)
	}

	if err := s.publishDevicesUnregistered(ctx, owner, userID, deletedDeviceIds); err != nil {
		log.Errorf("cannot publish devices unregistered event with devices('%v') and owner('%v'): %w", deletedDeviceIds, owner, err)
	}

	return &pb.DeleteDevicesResponse{
		DeviceIds: deletedDeviceIds,
	}, nil
}

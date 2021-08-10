package service

import (
	"context"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/plgd-dev/cloud/authorization/events"
	"github.com/plgd-dev/cloud/authorization/pb"
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

func (s *Service) publishDevicesUnregistered(ctx context.Context, owner string, deviceIDs []string) error {
	v := events.Event{
		Type: &events.Event_DevicesUnregistered{
			DevicesUnregistered: &events.DevicesUnregistered{
				Owner:     owner,
				DeviceIds: deviceIDs,
				AuditContext: &events.AuditContext{
					UserId: owner,
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

	owner := request.UserId
	if owner == "" {
		if token, err := grpc_auth.AuthFromMD(ctx, "bearer"); err == nil {
			uid, err := grpc.ParseOwnerFromJwtToken(s.ownerClaim, token)
			if err == nil {
				owner = uid
			}
		}
	}

	if owner == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete devices: invalid UserId"))
	}

	deviceIds := getUniqueDeviceIds(request.DeviceIds)
	if len(deviceIds) == 0 {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete devices: invalid DeviceIds"))
	}

	// TODO validate jwt token -> only jwt token is supported

	var deletedDeviceIds []string
	for _, deviceId := range request.DeviceIds {
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

	if err := s.publishDevicesUnregistered(ctx, owner, deletedDeviceIds); err != nil {
		log.Errorf("cannot publish devices unregistered event with devices('%v') and owner('%v'): %w", deletedDeviceIds, owner, err)
	}

	return &pb.DeleteDevicesResponse{
		DeviceIds: deletedDeviceIds,
	}, nil
}

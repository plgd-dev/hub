package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/identity-store/persistence"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry/propagation"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/kit/v2/strings"
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

func (s *Service) publishDevicesUnregistered(ctx context.Context, owner, userID, hubID string, deviceIDs []string) {
	v := events.Event{
		Type: &events.Event_DevicesUnregistered{
			DevicesUnregistered: &events.DevicesUnregistered{
				Owner:     owner,
				DeviceIds: deviceIDs,
				AuditContext: &events.AuditContext{
					UserId: userID,
				},
				Timestamp:            pkgTime.UnixNano(time.Now()),
				OpenTelemetryCarrier: propagation.TraceFromCtx(ctx),
				EventMetadata: &events.EventMetadata{
					HubId: hubID,
				},
			},
		},
	}

	subject := events.GetDevicesUnregisteredSubject(owner)
	err := s.publishEvent(subject, &v)
	publisher.LogPublish(s.logger, &v, []string{subject}, err)
}

func getDeviceIDs(request *pb.DeleteDevicesRequest, tx persistence.PersistenceTx, owner string) ([]string, error) {
	var deviceIds []string
	if len(request.DeviceIds) == 0 {
		var err error
		if deviceIds, err = getOwnerDevices(tx, owner); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot delete devices: %v", err)
		}
		return deviceIds, nil
	}
	deviceIds = getUniqueDeviceIds(request.DeviceIds)
	if len(deviceIds) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete devices: invalid DeviceIds")
	}
	return deviceIds, nil
}

func deleteDevice(tx persistence.PersistenceTx, deviceId, owner string) (bool, error) {
	_, ok, err := tx.Retrieve(deviceId, owner)
	if err != nil {
		return false, status.Errorf(codes.Internal, "cannot delete device('%v'): %v", deviceId, err.Error())
	}
	if !ok {
		log.Debugf("cannot retrieve device by user('%v')", owner)
		return false, nil
	}

	if err = tx.Delete(deviceId, owner); err != nil {
		return false, status.Errorf(codes.NotFound, "cannot delete device('%v'): %v", deviceId, err)
	}
	return true, nil
}

// DeleteDevices removes a devices from user.
func (s *Service) DeleteDevices(ctx context.Context, request *pb.DeleteDevicesRequest) (*pb.DeleteDevicesResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner, userID, err := parseTokenMD(ctx, s.ownerClaim)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardFromError(codes.InvalidArgument, fmt.Errorf("cannot delete devices: %w", err)))
	}

	deviceIDs, err := getDeviceIDs(request, tx, owner)
	if err != nil {
		return nil, log.LogAndReturnError(err)
	}
	if len(deviceIDs) == 0 {
		return &pb.DeleteDevicesResponse{}, nil
	}

	var deletedDeviceIDs []string
	for _, deviceID := range deviceIDs {
		ok, err := deleteDevice(tx, deviceID, owner)
		if err != nil {
			return nil, log.LogAndReturnError(err)
		}
		if !ok {
			continue
		}
		deletedDeviceIDs = append(deletedDeviceIDs, deviceID)
	}

	s.publishDevicesUnregistered(ctx, owner, userID, s.hubID, deletedDeviceIDs)

	return &pb.DeleteDevicesResponse{
		DeviceIds: deletedDeviceIDs,
	}, nil
}

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	return nil
}

// DeleteDevice removes a device from user. It is used by cloud2cloud connector.
func (s *Service) DeleteDevice(ctx context.Context, request *pb.DeleteDeviceRequest) (*pb.DeleteDeviceResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner := request.UserId
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		uid, err := grpc.ParseOwnerFromJwtToken(s.ownerClaim, token)
		if err == nil {
			owner = uid
		}
	}

	if owner == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot delete device: invalid DeviceId"))
	}

	// TODO validate jwt token -> only jwt token is supported

	_, ok, err := tx.Retrieve(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot delete device: %v", err.Error()))
	}
	if !ok {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot delete device: not found"))
	}

	err = tx.Delete(request.DeviceId, owner)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.NotFound, "cannot delete device: not found"))
	}

	err = s.publishDevicesUnregistered(ctx, owner, []string{request.DeviceId})
	if err != nil {
		log.Errorf("cannot publish devices unregistered event with device('%v') and owner('%v'): %w", request.DeviceId, owner, err)
	}

	return &pb.DeleteDeviceResponse{}, nil
}

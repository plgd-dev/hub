package service

import (
	"context"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/plgd-dev/cloud/authorization/events"
	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/authorization/persistence"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const serviceOwner = "*"

func (s *Service) publishDevicesRegistered(ctx context.Context, owner, sub string, deviceID []string) error {
	// TODO: verify that s.ownerClaim and sub are filled in JWToken
	//	- use s.ownerClaim value from JWT for DevicesUnregistered.Owner
	//	- use 'sub' from JWT for AuditContext
	v := events.Event{
		Type: &events.Event_DevicesRegistered{
			DevicesRegistered: &events.DevicesRegistered{
				Owner:     owner,
				DeviceIds: deviceID,
				AuditContext: &events.AuditContext{
					UserId: sub,
				},
				Timestamp: pkgTime.UnixNano(time.Now()),
			},
		},
	}
	data, err := utils.Marshal(&v)
	if err != nil {
		return err
	}

	err = s.publisher.PublishData(ctx, events.GetDevicesRegisteredSubject(owner), data)
	if err != nil {
		return err
	}

	err = s.publisher.Flush(ctx)
	if err != nil {
		return err
	}
	return nil
}

// AddDevice adds a device to user. It is used by cloud2cloud connector.
func (s *Service) AddDevice(ctx context.Context, request *pb.AddDeviceRequest) (*pb.AddDeviceResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner := request.UserId
	sub := owner
	// TODO: always use value from JWT, remove UserId from pb.AddDeviceRequest
	if token, err := grpc_auth.AuthFromMD(ctx, "bearer"); err == nil {
		// TODO: add mechanism to add token with correct data to tests, then uncomment the following line
		// return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot delete devices: %v", err))

		// TODO: check that both s.ownerClaim and "sub" are set in token
		if owner == "" {
			uid, err := grpc.ParseOwnerFromJwtToken(s.ownerClaim, token)
			if err == nil && uid != serviceOwner {
				owner = uid
			}
		}
		if sub == "" {
			uid, err := grpc.ParseOwnerFromJwtToken("sub", token)
			if err == nil {
				sub = uid
			}
		}
	}

	if owner == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: invalid UserId"))
	}

	if request.DeviceId == "" {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: invalid DeviceId"))
	}

	dev, ok, err := tx.RetrieveByDevice(request.DeviceId)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot add device: %v", err.Error()))
	}
	if ok {
		if dev.Owner == owner {
			return &pb.AddDeviceResponse{}, nil
		}
		return nil, log.LogAndReturnError(status.Errorf(codes.Unauthenticated, "cannot add device: devices is owned by another user '%+v'", dev))
	}

	d := persistence.AuthorizedDevice{
		DeviceID:     request.DeviceId,
		Owner:        owner,
		AccessToken:  "",
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	if err := tx.Persist(&d); err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot add device up: %v", err.Error()))
	}

	err = s.publishDevicesRegistered(ctx, owner, sub, []string{request.DeviceId})
	if err != nil {
		log.Errorf("cannot publish devices registered event with device('%v') and owner('%v'): %w", request.DeviceId, owner, err)
	}

	return &pb.AddDeviceResponse{}, nil
}

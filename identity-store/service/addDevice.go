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
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) publishEvent(subject string, event utils.ProtobufMarshaler) error {
	data, err := utils.Marshal(event)
	if err != nil {
		return err
	}
	err = s.publisher.PublishData(subject, data)
	if err != nil {
		return err
	}
	// timeout si driven by flusherTimeout.
	return s.publisher.Flush(context.Background())
}

func (s *Service) publishDevicesRegistered(ctx context.Context, owner, userID, hubID string, deviceID []string) {
	v := events.Event{
		Type: &events.Event_DevicesRegistered{
			DevicesRegistered: &events.DevicesRegistered{
				Owner:     owner,
				DeviceIds: deviceID,
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
	subject := events.GetDevicesRegisteredSubject(owner)
	err := s.publishEvent(subject, &v)
	publisher.LogPublish(s.logger, &v, []string{subject}, err)
}

func (s *Service) parseTokenMD(ctx context.Context) (owner, subject string, err error) {
	token, err := grpc.TokenFromMD(ctx)
	if err != nil {
		return "", "", grpc.ForwardFromError(codes.InvalidArgument, err)
	}
	claims, err := jwt.ParseToken(token)
	if err != nil {
		return "", "", grpc.ForwardFromError(codes.InvalidArgument, err)
	}
	owner = claims.Owner(s.ownerClaim)
	if owner == "" {
		return "", "", status.Errorf(codes.InvalidArgument, "%v", fmt.Errorf("claim '%v' was not found", s.ownerClaim))
	}
	subject = claims.Subject()
	if subject == "" {
		return "", "", status.Errorf(codes.InvalidArgument, "%v", fmt.Errorf("claim '%v' was not found", "sub"))
	}
	return
}

// AddDevice adds a device to user. It is used by cloud2cloud connector.
func (s *Service) AddDevice(ctx context.Context, request *pb.AddDeviceRequest) (*pb.AddDeviceResponse, error) {
	tx := s.persistence.NewTransaction(ctx)
	defer tx.Close()

	owner, userID, err := s.parseTokenMD(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: %v", err))
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
		DeviceID: request.DeviceId,
		Owner:    owner,
	}

	if err = tx.Persist(&d); err != nil {
		return nil, log.LogAndReturnError(status.Errorf(codes.Internal, "cannot add device up: %v", err.Error()))
	}

	s.publishDevicesRegistered(ctx, owner, userID, s.hubID, []string{request.DeviceId})

	return &pb.AddDeviceResponse{}, nil
}

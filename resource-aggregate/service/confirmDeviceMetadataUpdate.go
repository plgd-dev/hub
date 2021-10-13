package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validateConfirmDeviceMetadataUpdate(request *commands.ConfirmDeviceMetadataUpdateRequest) error {
	if request.GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	}
	if request.GetShadowSynchronization() == commands.ShadowSynchronization_UNSET {
		return status.Errorf(codes.InvalidArgument, "confirm.shadowSynchronizationStatus are invalid")
	}

	return nil
}

func (a *aggregate) ConfirmDeviceMetadataUpdate(ctx context.Context, request *commands.ConfirmDeviceMetadataUpdateRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmDeviceMetadataUpdate(request); err != nil {
		err = fmt.Errorf("invalid update device metadata command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process confirm device metadata update command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (r RequestHandler) ConfirmDeviceMetadataUpdate(ctx context.Context, request *commands.ConfirmDeviceMetadataUpdateRequest) (*commands.ConfirmDeviceMetadataUpdateResponse, error) {
	owner, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	resID := commands.NewResourceID(request.DeviceId, commands.StatusHref)
	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, DeviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm device('%v') metadata update: %v", request.GetDeviceId(), err))
	}

	events, err := aggregate.ConfirmDeviceMetadataUpdate(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot confirm device('%v') metadata update: %v", request.GetDeviceId(), err))
	}

	err = PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish device('%v') metadata events: %w", request.GetDeviceId(), err)
	}
	return &commands.ConfirmDeviceMetadataUpdateResponse{
		AuditContext: commands.NewAuditContext(owner, ""),
	}, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validateUpdateDeviceMetadata(request *commands.UpdateDeviceMetadataRequest) error {
	if err := checkTimeToLive(request.GetTimeToLive()); err != nil {
		return err
	}
	if request.GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	}
	switch v := request.GetUpdate().(type) {
	case *commands.UpdateDeviceMetadataRequest_Connection, *commands.UpdateDeviceMetadataRequest_TwinEnabled, *commands.UpdateDeviceMetadataRequest_TwinSynchronization, *commands.UpdateDeviceMetadataRequest_TwinForceSynchronization:
	default:
		return status.Errorf(codes.InvalidArgument, "update type (%T) invalid", v)
	}

	return nil
}

func (a *Aggregate) UpdateDeviceMetadata(ctx context.Context, request *commands.UpdateDeviceMetadataRequest) (events []eventstore.Event, err error) {
	if err = validateUpdateDeviceMetadata(request); err != nil {
		err = fmt.Errorf("invalid update device metadata command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update device metadata command command: %w", err)
		return
	}
	return
}

func (a *Aggregate) UpdateDeviceToOffline(ctx context.Context, request *events.UpdateDeviceToOfflineRequest) (events []eventstore.Event, err error) {
	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update device metadata command command: %w", err)
		return
	}
	return
}

func checkTimeToLiveForDefault(defaultTimeToLive time.Duration, reqTimeToLive int64) int64 {
	if defaultTimeToLive == 0 {
		return reqTimeToLive
	}
	if reqTimeToLive != 0 {
		return reqTimeToLive
	}
	return int64(defaultTimeToLive)
}

func (r RequestHandler) updateDeviceMetadata(ctx context.Context, request *commands.UpdateDeviceMetadataRequest, userID, owner string) (*commands.UpdateDeviceMetadataResponse, error) {
	request.TimeToLive = checkTimeToLiveForDefault(r.config.Clients.Eventstore.DefaultCommandTimeToLive, request.GetTimeToLive())

	resID := commands.NewResourceID(request.GetDeviceId(), commands.StatusHref)

	var latestSnapshot *events.DeviceMetadataSnapshotTakenForCommand
	deviceMetadataFactoryModel := func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		latestSnapshot = events.NewDeviceMetadataSnapshotTakenForCommand(userID, owner, r.config.HubID)
		return latestSnapshot, nil
	}

	aggregate, err := NewAggregate(resID, r.eventstore, deviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update device('%v') metadata: %v", request.GetDeviceId(), err))
	}

	publishEvents, err := aggregate.UpdateDeviceMetadata(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update device('%v') metadata: %v", request.GetDeviceId(), err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), publishEvents, r.logger)

	var validUntil int64
	for _, e := range publishEvents {
		if ev, ok := e.(*events.DeviceMetadataUpdatePending); ok {
			validUntil = ev.GetValidUntil()
			break
		}
	}

	return &commands.UpdateDeviceMetadataResponse{
		AuditContext: commands.NewAuditContext(userID, "", owner),
		TwinEnabled:  latestSnapshot.GetDeviceMetadataUpdated().GetTwinEnabled(),
		ValidUntil:   validUntil,
	}, nil
}

func (r RequestHandler) UpdateDeviceMetadata(ctx context.Context, request *commands.UpdateDeviceMetadataRequest) (*commands.UpdateDeviceMetadataResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	return r.updateDeviceMetadata(ctx, request, userID, owner)
}

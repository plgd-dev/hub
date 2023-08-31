package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validateCancelPendingMetadataUpdates(request *commands.CancelPendingMetadataUpdatesRequest) error {
	if request.GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	}

	return nil
}

func (a *Aggregate) CancelPendingMetadataUpdates(ctx context.Context, request *commands.CancelPendingMetadataUpdatesRequest) (events []eventstore.Event, err error) {
	if err = validateCancelPendingMetadataUpdates(request); err != nil {
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process cancel resource command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (r RequestHandler) CancelPendingMetadataUpdates(ctx context.Context, request *commands.CancelPendingMetadataUpdatesRequest) (*commands.CancelPendingMetadataUpdatesResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	resID := commands.NewResourceID(request.GetDeviceId(), commands.StatusHref)
	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewDeviceMetadataFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel device ('%v') metadata updates: %v", request.GetDeviceId(), err))
	}

	cancelEvents, err := aggregate.CancelPendingMetadataUpdates(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel resource('%v') metadata updates: %v", request.GetDeviceId(), err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), cancelEvents, r.logger)

	correlationIDs := make([]string, 0, len(cancelEvents))
	for _, e := range cancelEvents {
		if ev, ok := e.(*events.DeviceMetadataUpdated); ok {
			correlationIDs = append(correlationIDs, ev.GetAuditContext().GetCorrelationId())
		}
	}

	return &commands.CancelPendingMetadataUpdatesResponse{
		AuditContext:   commands.NewAuditContext(owner, "", owner),
		CorrelationIds: correlationIDs,
	}, nil
}

package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validateCancelResourceCommand(request *commands.CancelPendingCommandsRequest) error {
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid Href")
	}

	return nil
}

func (a *aggregate) CancelResourceCommand(ctx context.Context, request *commands.CancelPendingCommandsRequest) (events []eventstore.Event, err error) {
	if err = validateCancelResourceCommand(request); err != nil {
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process cancel resource command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (r RequestHandler) CancelPendingCommands(ctx context.Context, request *commands.CancelPendingCommandsRequest) (*commands.CancelPendingCommandsResponse, error) {
	owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	resID := request.GetResourceId()
	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel resource('%v') command: %v", request.GetResourceId().ToString(), err))
	}

	cancelEvents, err := aggregate.CancelResourceCommand(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel resource('%v') command: %v", request.GetResourceId().ToString(), err))
	}

	err = PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), cancelEvents)
	if err != nil {
		log.Errorf("cannot publish resource('%v') events: %w", request.GetResourceId().ToString(), err)
	}

	correlationIDs := make([]string, 0, len(cancelEvents))
	for _, e := range cancelEvents {
		switch ev := e.(type) {
		case *events.ResourceCreated:
			correlationIDs = append(correlationIDs, ev.GetAuditContext().GetCorrelationId())
		case *events.ResourceUpdated:
			correlationIDs = append(correlationIDs, ev.GetAuditContext().GetCorrelationId())
		case *events.ResourceRetrieved:
			correlationIDs = append(correlationIDs, ev.GetAuditContext().GetCorrelationId())
		case *events.ResourceDeleted:
			correlationIDs = append(correlationIDs, ev.GetAuditContext().GetCorrelationId())
		}
	}

	return &commands.CancelPendingCommandsResponse{
		AuditContext:   commands.NewAuditContext(owner, ""),
		CorrelationIds: correlationIDs,
	}, nil
}

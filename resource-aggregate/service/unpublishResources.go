package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
)

func (a *aggregate) UnpublishResource(ctx context.Context, request *commands.UnpublishResourceLinksRequest) (events []eventstore.Event, err error) {
	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process unpublish resource command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)
	return
}

func (r RequestHandler) UnpublishResource(ctx context.Context, request *commands.UnpublishResourceLinksRequest, owner string, resourceID *commands.ResourceId) error {
	aggregate, err := NewAggregate(resourceID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return err
	}

	events, err := aggregate.UnpublishResource(ctx, request)
	if err != nil {
		return err
	}

	err = PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		return fmt.Errorf("cannot publish events to eventbus: %w", err)
	}
	return nil
}

func (r RequestHandler) UnpublishResources(ctx context.Context, request *commands.UnpublishResourceLinksRequest, owner string, events []eventstore.Event) {
	for _, event := range events {
		if rlu, ok := event.(*raEvents.ResourceLinksUnpublished); ok {
			for _, href := range rlu.GetHrefs() {
				err := r.UnpublishResource(ctx, request, owner, commands.NewResourceID(rlu.GetDeviceId(), href))
				if err != nil {
					_ = log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot unpublish resource /%v%v: %v", rlu.GetDeviceId(), href, err))
				}
			}
		}
	}
}

package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/log"
	"google.golang.org/grpc/codes"
)

type LogPublishErrFunc func(err error)

type aggregate struct {
	ag         *cqrsAggregate.Aggregate
	resourceID string
	eventstore EventStore
}

func resourceStateFactoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	return raEvents.NewResourceStateSnapshotTaken(), nil
}

func resourceLinksFactoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	return raEvents.NewResourceLinksSnapshotTaken(), nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(resourceID *commands.ResourceId, SnapshotThreshold int, eventstore EventStore, factoryModel cqrsAggregate.FactoryModelFunc, retry cqrsAggregate.RetryFunc) (*aggregate, error) {
	a := &aggregate{
		eventstore: eventstore,
	}
	cqrsAg, err := cqrsAggregate.NewAggregate(resourceID.GetDeviceId(), resourceID.ToUUID(), retry, SnapshotThreshold, eventstore, factoryModel, func(template string, args ...interface{}) {})
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resource: %w", err)
	}
	a.ag = cqrsAg
	return a, nil
}

func validatePublish(request *commands.PublishResourceLinksRequest) error {
	if len(request.Resources) == 0 {
		return status.Errorf(codes.InvalidArgument, "empty publish is not accepted")
	}
	for _, res := range request.Resources {
		if res.Href == "" {
			return status.Errorf(codes.InvalidArgument, "invalid resource href")
		}
	}
	if request.DeviceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid deviceID")
	}
	return nil
}

func validateUnpublish(request *commands.UnpublishResourceLinksRequest) error {
	if request.GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid deviceID")
	}
	for _, href := range request.Hrefs {
		if href == "" {
			return status.Errorf(codes.InvalidArgument, "invalid resource id")
		}
	}
	return nil
}

func validateNotifyContentChanged(request *commands.NotifyResourceChangedRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	return nil
}

func validateUpdateResourceContent(request *commands.UpdateResourceRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}
	return nil
}

func validateRetrieveResource(request *commands.RetrieveResourceRequest) error {
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}
	return nil
}

func validateConfirmResourceUpdate(request *commands.ConfirmResourceUpdateRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}

	return nil
}

func validateConfirmResourceRetrieve(request *commands.ConfirmResourceRetrieveRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}

	return nil
}

func validateDeleteResource(request *commands.DeleteResourceRequest) error {
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}
	return nil
}

func validateConfirmResourceDelete(request *commands.ConfirmResourceDeleteRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	}
	if request.GetResourceId().GetHref() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}

	return nil
}

func cleanUpToSnapshot(ctx context.Context, aggregate *aggregate, events []eventstore.Event) {
	for _, event := range events {
		if ru, ok := event.(*raEvents.ResourceStateSnapshotTaken); ok {
			if err := aggregate.eventstore.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: ru.GroupId(), AggregateID: ru.AggregateId(), Version: ru.Version()}}); err != nil {
				log.Info("unable to remove events up to snapshot for resource: %v", ru.GetResourceId())
			}
			break
		}
	}
}

func insertMaintenanceDbRecord(ctx context.Context, aggregate *aggregate, events []eventstore.Event) {
	for _, event := range events {
		if ru, ok := event.(*raEvents.ResourceStateSnapshotTaken); ok {
			if err := aggregate.eventstore.Insert(ctx, maintenance.Task{AggregateID: ru.AggregateId(), Version: ru.Version()}); err != nil {
				log.Info("unable to insert the snapshot information into the maintenance db")
			}
			break
		}
	}
}

func (a *aggregate) DeviceID() string {
	return a.ag.DeviceID()
}

// HandlePublishResource handles a command PublishResourceLinks
func (a *aggregate) PublishResourceLinks(ctx context.Context, request *commands.PublishResourceLinksRequest) (events []eventstore.Event, err error) {
	if err = validatePublish(request); err != nil {
		err = fmt.Errorf("invalid publish resource links command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process publish resource links command: %w", err)
		return
	}

	cleanUpToSnapshot(ctx, a, events)

	return
}

// HandleUnpublishResource handles a command UnpublishResourceLinks
func (a *aggregate) UnpublishResourceLinks(ctx context.Context, request *commands.UnpublishResourceLinksRequest) (events []eventstore.Event, err error) {
	if err = validateUnpublish(request); err != nil {
		err = fmt.Errorf("invalid unpublish resource links command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process unpublish resource links command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

// NotifyContentChanged handles a command NotifyContentChanged
func (a *aggregate) NotifyResourceChanged(ctx context.Context, request *commands.NotifyResourceChangedRequest) (events []eventstore.Event, err error) {
	if err = validateNotifyContentChanged(request); err != nil {
		err = fmt.Errorf("invalid notify content changed command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process notify content changed command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)
	return
}

// HandleUpdateResourceContent handles a command UpdateResource
func (a *aggregate) UpdateResource(ctx context.Context, request *commands.UpdateResourceRequest) (events []eventstore.Event, err error) {
	if err = validateUpdateResourceContent(request); err != nil {
		err = fmt.Errorf("invalid update resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (a *aggregate) ConfirmResourceUpdate(ctx context.Context, request *commands.ConfirmResourceUpdateRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceUpdate(request); err != nil {
		err = fmt.Errorf("invalid update resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content notification command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

// RetrieveResource handles a command RetriveResource
func (a *aggregate) RetrieveResource(ctx context.Context, request *commands.RetrieveResourceRequest) (events []eventstore.Event, err error) {
	if err = validateRetrieveResource(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (a *aggregate) ConfirmResourceRetrieve(ctx context.Context, request *commands.ConfirmResourceRetrieveRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceRetrieve(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content notification command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)
	return
}

// DeleteResource handles a command DeleteResource
func (a *aggregate) DeleteResource(ctx context.Context, request *commands.DeleteResourceRequest) (events []eventstore.Event, err error) {
	if err = validateDeleteResource(request); err != nil {
		err = fmt.Errorf("invalid delete resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (a *aggregate) ConfirmResourceDelete(ctx context.Context, request *commands.ConfirmResourceDeleteRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceDelete(request); err != nil {
		err = fmt.Errorf("invalid delete resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content notification command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

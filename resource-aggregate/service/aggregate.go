package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"

	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
	"google.golang.org/grpc/codes"
)

type LogPublishErrFunc func(err error)

type aggregate struct {
	model      *raEvents.ResourceStateSnapshotTaken
	ag         *cqrsAggregate.Aggregate
	resourceID string
	eventstore EventStore
}

func (a *aggregate) factoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	a.model = raEvents.NewResourceStateSnapshotTaken()
	return a.model, nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(resourceID *pb.ResourceId, SnapshotThreshold int, eventstore EventStore, retry cqrsAggregate.RetryFunc) (*aggregate, error) {
	resID := utils.MakeResourceId(resourceID.GetDeviceId(), resourceID.GetHref())
	a := &aggregate{
		resourceID: resID,
		eventstore: eventstore,
	}
	cqrsAg, err := cqrsAggregate.NewAggregate(resourceID.GetDeviceId(), resID, retry, SnapshotThreshold, eventstore, a.factoryModel, func(template string, args ...interface{}) {})
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resource: %w", err)
	}
	a.ag = cqrsAg
	return a, nil
}

func validatePublish(request *pb.PublishResourceLinksRequest) error {
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

func validateUnpublish(request *pb.UnpublishResourceLinksRequest) error {
	if request.GetDeviceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid deviceID")
	}
	for _, id := range request.Ids {
		if id == "" {
			return status.Errorf(codes.InvalidArgument, "invalid resource id")
		}
	}
	return nil
}

func validateNotifyContentChanged(request *pb.NotifyResourceChangedRequest) error {
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

func validateUpdateResourceContent(request *pb.UpdateResourceRequest) error {
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

func validateRetrieveResource(request *pb.RetrieveResourceRequest) error {
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

func validateConfirmResourceUpdate(request *pb.ConfirmResourceUpdateRequest) error {
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

func validateConfirmResourceRetrieve(request *pb.ConfirmResourceRetrieveRequest) error {
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

func validateDeleteResource(request *pb.DeleteResourceRequest) error {
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

func validateConfirmResourceDelete(request *pb.ConfirmResourceDeleteRequest) error {
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
				log.Info("unable to remove events up to snapshot /%v%v", ru.GetResource().GetDeviceId(), ru.GetResource().GetHref())
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
	return a.model.GroupId()
}

// HandlePublishResource handles a command PublishResourceLinks
func (a *aggregate) PublishResourceLinks(ctx context.Context, request *pb.PublishResourceLinksRequest) (events []eventstore.Event, err error) {
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
func (a *aggregate) UnpublishResourceLinks(ctx context.Context, request *pb.UnpublishResourceLinksRequest) (events []eventstore.Event, err error) {
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
func (a *aggregate) NotifyResourceChanged(ctx context.Context, request *pb.NotifyResourceChangedRequest) (events []eventstore.Event, err error) {
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
func (a *aggregate) UpdateResource(ctx context.Context, request *pb.UpdateResourceRequest) (events []eventstore.Event, err error) {
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

func (a *aggregate) ConfirmResourceUpdate(ctx context.Context, request *pb.ConfirmResourceUpdateRequest) (events []eventstore.Event, err error) {
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
func (a *aggregate) RetrieveResource(ctx context.Context, request *pb.RetrieveResourceRequest) (events []eventstore.Event, err error) {
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

func (a *aggregate) ConfirmResourceRetrieve(ctx context.Context, request *pb.ConfirmResourceRetrieveRequest) (events []eventstore.Event, err error) {
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
func (a *aggregate) DeleteResource(ctx context.Context, request *pb.DeleteResourceRequest) (events []eventstore.Event, err error) {
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

func (a *aggregate) ConfirmResourceDelete(ctx context.Context, request *pb.ConfirmResourceDeleteRequest) (events []eventstore.Event, err error) {
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

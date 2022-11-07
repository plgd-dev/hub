package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogPublishErrFunc func(err error)

type aggregate struct {
	ag         *cqrsAggregate.Aggregate
	eventstore EventStore
}

func ResourceStateFactoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	return raEvents.NewResourceStateSnapshotTaken(), nil
}

func ResourceLinksFactoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	return raEvents.NewResourceLinksSnapshotTaken(), nil
}

func DeviceMetadataFactoryModel(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
	return raEvents.NewDeviceMetadataSnapshotTaken(), nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(resourceID *commands.ResourceId, snapshotThreshold int, eventstore EventStore, factoryModel cqrsAggregate.FactoryModelFunc, retry cqrsAggregate.RetryFunc) (*aggregate, error) {
	a := &aggregate{
		eventstore: eventstore,
	}
	cqrsAg, err := cqrsAggregate.NewAggregate(resourceID.GetDeviceId(),
		resourceID.ToUUID(),
		retry,
		snapshotThreshold,
		eventstore,
		factoryModel,
		func(template string, args ...interface{}) {})
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resource: %w", err)
	}
	a.ag = cqrsAg
	return a, nil
}

var (
	errInvalidDeviceID           = status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	errInvalidContent            = status.Errorf(codes.InvalidArgument, "invalid Content")
	errInvalidCorrelationID      = status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	errInvalidResourceIdDeviceID = status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	errInvalidResourceIdHref     = status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
)

func validatePublish(request *commands.PublishResourceLinksRequest) error {
	if len(request.Resources) == 0 {
		return status.Errorf(codes.InvalidArgument, "empty publish is not accepted")
	}
	for _, res := range request.Resources {
		if len(res.Href) <= 1 || res.Href[:1] != "/" {
			return status.Errorf(codes.InvalidArgument, "invalid resource href")
		}
		if res.DeviceId == "" {
			return status.Errorf(codes.InvalidArgument, "invalid device id")
		}
	}
	if request.DeviceId == "" {
		return errInvalidDeviceID
	}
	return nil
}

func validateUnpublish(request *commands.UnpublishResourceLinksRequest) error {
	if request.GetDeviceId() == "" {
		return errInvalidDeviceID
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
		return errInvalidContent
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	return nil
}

func validateUpdateResourceContent(request *commands.UpdateResourceRequest) error {
	if err := checkTimeToLive(request.GetTimeToLive()); err != nil {
		return err
	}
	if request.Content == nil {
		return errInvalidContent
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}
	return nil
}

func validateRetrieveResource(request *commands.RetrieveResourceRequest) error {
	if err := checkTimeToLive(request.GetTimeToLive()); err != nil {
		return err
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}
	return nil
}

func validateConfirmResourceUpdate(request *commands.ConfirmResourceUpdateRequest) error {
	if request.Content == nil {
		return errInvalidContent
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}

	return nil
}

func validateConfirmResourceRetrieve(request *commands.ConfirmResourceRetrieveRequest) error {
	if request.Content == nil {
		return errInvalidContent
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}

	return nil
}

func validateDeleteResource(request *commands.DeleteResourceRequest) error {
	if err := checkTimeToLive(request.GetTimeToLive()); err != nil {
		return err
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}
	return nil
}

func validateCreateResource(request *commands.CreateResourceRequest) error {
	if err := checkTimeToLive(request.GetTimeToLive()); err != nil {
		return err
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}
	if request.GetContent() == nil {
		return errInvalidContent
	}
	if request.GetContent().GetData() == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content.Data")
	}
	return nil
}

func validateConfirmResourceCreate(request *commands.ConfirmResourceCreateRequest) error {
	if request.GetContent() == nil {
		return errInvalidContent
	}
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}

	return nil
}

func validateConfirmResourceDelete(request *commands.ConfirmResourceDeleteRequest) error {
	if request.GetResourceId().GetDeviceId() == "" {
		return errInvalidResourceIdDeviceID
	}
	if request.GetResourceId().GetHref() == "" {
		return errInvalidResourceIdHref
	}
	if request.GetCorrelationId() == "" {
		return errInvalidCorrelationID
	}

	return nil
}

func cleanUpToSnapshot(ctx context.Context, aggregate *aggregate, events []eventstore.Event) {
	for _, event := range events {
		if event.IsSnapshot() {
			err := aggregate.eventstore.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: event.GroupID(), AggregateID: event.AggregateID(), Version: event.Version()}})
			if err != nil {
				if ru, ok := event.(interface{ GetResourceId() *commands.ResourceId }); ok {
					log.Info("unable to remove events up to snapshot with version('%v') for resource('%v')", event.Version(), ru.GetResourceId())
				} else {
					log.Info("unable to remove events up to snapshot(%v) with version('%v') of deviceId('%v')", event.EventType(), event.Version(), event.GroupID())
				}
			}
			break
		}
	}
}

func (a *aggregate) DeviceID() string {
	return a.ag.GroupID()
}

func (a *aggregate) ResourceID() string {
	return a.ag.AggregateID()
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

// CreateResource handles a command CreateResource
func (a *aggregate) CreateResource(ctx context.Context, request *commands.CreateResourceRequest) (events []eventstore.Event, err error) {
	if err = validateCreateResource(request); err != nil {
		err = fmt.Errorf("invalid create resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process create resource content command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

func (a *aggregate) ConfirmResourceCreate(ctx context.Context, request *commands.ConfirmResourceCreateRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceCreate(request); err != nil {
		err = fmt.Errorf("invalid create resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process create resource content notification command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)

	return
}

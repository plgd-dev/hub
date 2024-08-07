package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogPublishErrFunc func(err error)

type Aggregate struct {
	ag         *cqrsAggregate.Aggregate
	eventstore eventstore.EventStore
}

type resourceStateModel struct {
	resourceState *events.ResourceStateSnapshotTakenForCommand
	resourceLinks *events.ResourceLinksSnapshotTakenForCommand
}

func newResourceStateModel(userID, owner, hubID string) *resourceStateModel {
	resourceLinks := events.NewResourceLinksSnapshotTakenForCommand(userID, owner, hubID)
	resourceState := events.NewResourceStateSnapshotTakenForCommand(userID, owner, hubID, resourceLinks)
	return &resourceStateModel{
		resourceState: resourceState,
		resourceLinks: resourceLinks,
	}
}

func (r *resourceStateModel) isPublished(resourceID *commands.ResourceId) bool {
	if r.resourceLinks == nil {
		return false
	}
	if r.resourceLinks.GetResources() == nil {
		return false
	}
	return r.resourceLinks.GetResources()[resourceID.GetHref()] != nil
}

func (r *resourceStateModel) model(_ context.Context, groupID string, aggregateID string) (cqrsAggregate.AggregateModel, error) {
	resID := commands.NewResourceID(groupID, commands.ResourceLinksHref)
	if aggregateID == resID.ToUUID().String() {
		return r.resourceLinks, nil
	}
	return r.resourceState, nil
}

func NewResourceStateFactoryModel(userID, owner, hubID string) func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
	m := newResourceStateModel(userID, owner, hubID)
	return m.model
}

func NewResourceLinksFactoryModel(userID, owner, hubID string) func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
	return func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		return events.NewResourceLinksSnapshotTakenForCommand(userID, owner, hubID), nil
	}
}

func NewDeviceMetadataFactoryModel(userID, owner, hubID string) func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
	return func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		return events.NewDeviceMetadataSnapshotTakenForCommand(userID, owner, hubID), nil
	}
}

func NewServicesMetadataFactoryModel(userID, owner, hubID string) func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
	return func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		return events.NewServiceMetadataSnapshotTakenForCommand(userID, owner, hubID), nil
	}
}

// NewResourceAggregate for creating new resource aggregate.
func NewResourceAggregate(resourceID *commands.ResourceId, store eventstore.EventStore, factoryModel cqrsAggregate.FactoryModelFunc, retry cqrsAggregate.RetryFunc, addLinkedResources bool) (*Aggregate, error) {
	a := &Aggregate{
		eventstore: store,
	}
	addLink := make([]cqrsAggregate.AdditionalModel, 0, 1)
	if addLinkedResources {
		addLink = append(addLink, cqrsAggregate.AdditionalModel{
			GroupID:     resourceID.GetDeviceId(),
			AggregateID: commands.NewResourceID(resourceID.GetDeviceId(), commands.ResourceLinksHref).ToUUID().String(),
		})
	}

	cqrsAg, err := cqrsAggregate.NewAggregate(resourceID.GetDeviceId(),
		resourceID.ToUUID().String(),
		retry,
		store,
		factoryModel,
		func(string, ...interface{}) {
			// no-op - we don't want to log debug/trace messages
		},
		// load also links state
		addLink...,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resource: %w", err)
	}
	a.ag = cqrsAg
	return a, nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(resourceID *commands.ResourceId, store eventstore.EventStore, factoryModel cqrsAggregate.FactoryModelFunc, retry cqrsAggregate.RetryFunc) (*Aggregate, error) {
	return NewResourceAggregate(resourceID, store, factoryModel, retry, false)
}

func (a *Aggregate) HandleCommand(ctx context.Context, cmd cqrsAggregate.Command) ([]eventstore.Event, error) {
	events, err := a.ag.HandleCommand(ctx, cmd)
	if err == nil {
		a.cleanUpToSnapshot(ctx, events)
	}
	return events, err
}

var (
	errInvalidDeviceID           = status.Errorf(codes.InvalidArgument, "invalid DeviceId")
	errInvalidContent            = status.Errorf(codes.InvalidArgument, "invalid Content")
	errInvalidCorrelationID      = status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	errInvalidResourceIdDeviceID = status.Errorf(codes.InvalidArgument, "invalid ResourceId.DeviceId")
	errInvalidResourceIdHref     = status.Errorf(codes.InvalidArgument, "invalid ResourceId.Href")
)

func validatePublish(request *commands.PublishResourceLinksRequest) error {
	resources := request.GetResources()
	if len(resources) == 0 {
		return status.Errorf(codes.InvalidArgument, "empty publish is not accepted")
	}
	for _, res := range resources {
		href := res.GetHref()
		if len(href) <= 1 || href[:1] != "/" {
			return status.Errorf(codes.InvalidArgument, "invalid resource href")
		}
		if res.GetDeviceId() == "" {
			return status.Errorf(codes.InvalidArgument, "invalid device id")
		}
	}
	if request.GetDeviceId() == "" {
		return errInvalidDeviceID
	}
	return nil
}

func validateUnpublish(request *commands.UnpublishResourceLinksRequest) error {
	if request.GetDeviceId() == "" {
		return errInvalidDeviceID
	}
	for _, href := range request.GetHrefs() {
		if href == "" {
			return status.Errorf(codes.InvalidArgument, "invalid resource id")
		}
	}
	return nil
}

func validateNotifyContentChanged(request *commands.NotifyResourceChangedRequest) error {
	if request.GetContent() == nil {
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

func validateConfirmResourceRetrieve(request *commands.ConfirmResourceRetrieveRequest) error {
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

// cleanUpToSnapshot removes events up to latest snapshot, the database can contains only snapshots if it is supports multiple snapshots.
func (a *Aggregate) cleanUpToSnapshot(ctx context.Context, events []eventstore.Event) {
	cleanUp := make(map[string]eventstore.Event)
	// deduplicate events
	for _, event := range events {
		e, ok := cleanUp[event.AggregateID()]
		if !ok || e.Version() < event.Version() {
			cleanUp[event.AggregateID()] = event
		}
	}
	for _, event := range cleanUp {
		err := a.eventstore.RemoveUpToVersion(ctx, []eventstore.VersionQuery{{GroupID: event.GroupID(), AggregateID: event.AggregateID(), Version: event.Version()}})
		if err != nil && !errors.Is(err, eventstore.ErrNotSupported) {
			if ru, ok := event.(interface{ GetResourceId() *commands.ResourceId }); ok {
				log.Infof("unable to remove events up to snapshot with version('%v') for resource('%v')", event.Version(), ru.GetResourceId())
			} else {
				log.Infof("unable to remove events up to snapshot(%v) with version('%v') of deviceId('%v')", event.EventType(), event.Version(), event.GroupID())
			}
		}
	}
}

func (a *Aggregate) DeviceID() string {
	return a.ag.GroupID()
}

func (a *Aggregate) ResourceID() string {
	return a.ag.AggregateID()
}

// HandlePublishResource handles a command PublishResourceLinks
func (a *Aggregate) PublishResourceLinks(ctx context.Context, request *commands.PublishResourceLinksRequest) (events []eventstore.Event, err error) {
	if err = validatePublish(request); err != nil {
		err = fmt.Errorf("invalid publish resource links command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process publish resource links command: %w", err)
		return
	}

	return
}

// HandleUnpublishResource handles a command UnpublishResourceLinks
func (a *Aggregate) UnpublishResourceLinks(ctx context.Context, request *commands.UnpublishResourceLinksRequest) (events []eventstore.Event, err error) {
	if err = validateUnpublish(request); err != nil {
		err = fmt.Errorf("invalid unpublish resource links command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process unpublish resource links command: %w", err)
		return
	}
	return
}

// NotifyContentChanged handles a command NotifyContentChanged
func (a *Aggregate) NotifyResourceChanged(ctx context.Context, request *commands.NotifyResourceChangedRequest) (events []eventstore.Event, err error) {
	if err = validateNotifyContentChanged(request); err != nil {
		err = fmt.Errorf("invalid notify content changed command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process notify content changed command: %w", err)
		return
	}
	return
}

// HandleUpdateResourceContent handles a command UpdateResource
func (a *Aggregate) UpdateResource(ctx context.Context, request *commands.UpdateResourceRequest) (events []eventstore.Event, err error) {
	if err = validateUpdateResourceContent(request); err != nil {
		err = fmt.Errorf("invalid update resource content command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content command: %w", err)
		return
	}
	return
}

func (a *Aggregate) ConfirmResourceUpdate(ctx context.Context, request *commands.ConfirmResourceUpdateRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceUpdate(request); err != nil {
		err = fmt.Errorf("invalid update resource content notification command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content notification command: %w", err)
		return
	}
	return
}

// RetrieveResource handles a command RetriveResource
func (a *Aggregate) RetrieveResource(ctx context.Context, request *commands.RetrieveResourceRequest) (events []eventstore.Event, err error) {
	if err = validateRetrieveResource(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content command: %w", err)
		return
	}
	return
}

func (a *Aggregate) ConfirmResourceRetrieve(ctx context.Context, request *commands.ConfirmResourceRetrieveRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceRetrieve(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content notification command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content notification command: %w", err)
		return
	}
	return
}

// DeleteResource handles a command DeleteResource
func (a *Aggregate) DeleteResource(ctx context.Context, request *commands.DeleteResourceRequest) (events []eventstore.Event, err error) {
	if err = validateDeleteResource(request); err != nil {
		err = fmt.Errorf("invalid delete resource content command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content command: %w", err)
		return
	}
	return
}

func (a *Aggregate) ConfirmResourceDelete(ctx context.Context, request *commands.ConfirmResourceDeleteRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceDelete(request); err != nil {
		err = fmt.Errorf("invalid delete resource content notification command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content notification command: %w", err)
		return
	}
	return
}

// CreateResource handles a command CreateResource
func (a *Aggregate) CreateResource(ctx context.Context, request *commands.CreateResourceRequest) (events []eventstore.Event, err error) {
	if err = validateCreateResource(request); err != nil {
		err = fmt.Errorf("invalid create resource content command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process create resource content command: %w", err)
		return
	}
	return
}

func (a *Aggregate) ConfirmResourceCreate(ctx context.Context, request *commands.ConfirmResourceCreateRequest) (events []eventstore.Event, err error) {
	if err = validateConfirmResourceCreate(request); err != nil {
		err = fmt.Errorf("invalid create resource content notification command: %w", err)
		return
	}

	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process create resource content notification command: %w", err)
		return
	}
	return
}

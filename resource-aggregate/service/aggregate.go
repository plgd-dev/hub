package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"

	cqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/cqrs/events"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	cqrs "github.com/plgd-dev/cqrs"
	cqrsEvent "github.com/plgd-dev/cqrs/event"
	"github.com/plgd-dev/cqrs/eventstore/maintenance"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

type LogPublishErrFunc func(err error)
type aggregate struct {
	model            *raEvents.ResourceStateSnapshotTaken
	ag               *cqrs.Aggregate
	resourceId       string
	isUserDeviceFunc isUserDeviceFunc
	eventstore       EventStore
	userID           string
}

func (a *aggregate) factoryModel(ctx context.Context) (cqrs.AggregateModel, error) {
	a.model = raEvents.NewResourceStateSnapshotTaken(func(deviceId, resourceId string) error {
		ok, err := a.isUserDeviceFunc(ctx, a.userID, deviceId)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		return fmt.Errorf("access denied")
	})
	return a.model, nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(ctx context.Context, resourceId string, isUserDeviceFunc isUserDeviceFunc, SnapshotThreshold int, eventstore EventStore, retry cqrs.RetryFunc) (*aggregate, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resourced: invalid userID: %w", err)
	}

	a := &aggregate{
		resourceId:       resourceId,
		isUserDeviceFunc: isUserDeviceFunc,
		eventstore:       eventstore,
		userID:           userID,
	}
	cqrsAg, err := cqrs.NewAggregate(resourceId, retry, SnapshotThreshold, eventstore, a.factoryModel, func(template string, args ...interface{}) {})
	if err != nil {
		return nil, fmt.Errorf("cannot create aggregate for resource: %w", err)
	}
	a.ag = cqrsAg
	return a, nil
}

func validatePublish(request *pb.PublishResourceRequest) error {
	if request.Resource == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Resource")
	}
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	if request.Resource.Id != request.ResourceId {
		return status.Errorf(codes.InvalidArgument, "invalid Resource.Id")
	}
	if request.Resource.DeviceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid Resource.DeviceId")
	}
	return nil
}

func validateUnpublish(request *pb.UnpublishResourceRequest) error {
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	return nil
}

func validateNotifyContentChanged(request *pb.NotifyResourceChangedRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	return nil
}

func validateUpdateResourceContent(request *pb.UpdateResourceRequest) error {
	if request.Content == nil {
		return status.Errorf(codes.InvalidArgument, "invalid Content")
	}
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}
	return nil
}

func validateRetrieveResource(request *pb.RetrieveResourceRequest) error {
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
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
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
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
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}

	return nil
}

func validateDeleteResource(request *pb.DeleteResourceRequest) error {
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
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
	if request.ResourceId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid ResourceId")
	}
	if request.CorrelationId == "" {
		return status.Errorf(codes.InvalidArgument, "invalid CorrelationId")
	}

	return nil
}

func insertMaintenanceDbRecord(ctx context.Context, aggregate *aggregate, events []cqrsEvent.Event) {
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

// HandlePublishResource handles a command PublishResource
func (a *aggregate) PublishResource(ctx context.Context, request *pb.PublishResourceRequest) (response *pb.PublishResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validatePublish(request); err != nil {
		err = fmt.Errorf("invalid publish command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process publish command: %w", err)
		return
	}

	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, "")
	response = &pb.PublishResourceResponse{
		AuditContext: &auditContext,
	}
	return
}

// HandleUnpublishResource handles a command UnpublishResource
func (a *aggregate) UnpublishResource(ctx context.Context, request *pb.UnpublishResourceRequest) (response *pb.UnpublishResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validateUnpublish(request); err != nil {
		err = fmt.Errorf("invalid unpublish command: %w", err)
		return
	}
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		err = status.Errorf(codes.InvalidArgument, "cannot process unpublish command: invalid userID: %v", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process unpublish command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	response = &pb.UnpublishResourceResponse{
		AuditContext: &auditContext,
	}
	return
}

// NotifyContentChanged handles a command NotifyContentChanged
func (a *aggregate) NotifyResourceChanged(ctx context.Context, request *pb.NotifyResourceChangedRequest) (response *pb.NotifyResourceChangedResponse, events []cqrsEvent.Event, err error) {
	if err = validateNotifyContentChanged(request); err != nil {
		err = fmt.Errorf("invalid notify content changed command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process notify content changed command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, "")
	response = &pb.NotifyResourceChangedResponse{
		AuditContext: &auditContext,
	}
	return
}

// HandleUpdateResourceContent handles a command UpdateResource
func (a *aggregate) UpdateResource(ctx context.Context, request *pb.UpdateResourceRequest) (response *pb.UpdateResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validateUpdateResourceContent(request); err != nil {
		err = fmt.Errorf("invalid update resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.UpdateResourceResponse{
		AuditContext: &auditContext,
	}
	return
}

func (a *aggregate) ConfirmResourceUpdate(ctx context.Context, request *pb.ConfirmResourceUpdateRequest) (response *pb.ConfirmResourceUpdateResponse, events []cqrsEvent.Event, err error) {
	if err = validateConfirmResourceUpdate(request); err != nil {
		err = fmt.Errorf("invalid update resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update resource content notification command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.ConfirmResourceUpdateResponse{
		AuditContext: &auditContext,
	}
	return
}

// RetrieveResource handles a command RetriveResource
func (a *aggregate) RetrieveResource(ctx context.Context, request *pb.RetrieveResourceRequest) (response *pb.RetrieveResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validateRetrieveResource(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.RetrieveResourceResponse{
		AuditContext: &auditContext,
	}
	return
}

func (a *aggregate) ConfirmResourceRetrieve(ctx context.Context, request *pb.ConfirmResourceRetrieveRequest) (response *pb.ConfirmResourceRetrieveResponse, events []cqrsEvent.Event, err error) {
	if err = validateConfirmResourceRetrieve(request); err != nil {
		err = fmt.Errorf("invalid retrieve resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process retrieve resource content notification command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.ConfirmResourceRetrieveResponse{
		AuditContext: &auditContext,
	}
	return
}

// DeleteResource handles a command DeleteResource
func (a *aggregate) DeleteResource(ctx context.Context, request *pb.DeleteResourceRequest) (response *pb.DeleteResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validateDeleteResource(request); err != nil {
		err = fmt.Errorf("invalid delete resource content command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.DeleteResourceResponse{
		AuditContext: &auditContext,
	}
	return
}

func (a *aggregate) ConfirmResourceDelete(ctx context.Context, request *pb.ConfirmResourceDeleteRequest) (response *pb.ConfirmResourceDeleteResponse, events []cqrsEvent.Event, err error) {
	if err = validateConfirmResourceDelete(request); err != nil {
		err = fmt.Errorf("invalid delete resource content notification command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process delete resource content notification command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), a.userID, request.CorrelationId)
	response = &pb.ConfirmResourceDeleteResponse{
		AuditContext: &auditContext,
	}
	return
}

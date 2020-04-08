package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"

	cqrsUtils "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	"github.com/go-ocf/cloud/resource-aggregate/pb"
	cqrs "github.com/go-ocf/cqrs"
	cqrsEvent "github.com/go-ocf/cqrs/event"
	"github.com/go-ocf/cqrs/eventstore/maintenance"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"
)

type LogPublishErrFunc func(err error)
type aggregate struct {
	model         *raEvents.ResourceStateSnapshotTaken
	ag            *cqrs.Aggregate
	resourceId    string
	userDeviceIds strings.Set
	eventstore    EventStore
}

func (a *aggregate) factoryModel(context.Context) (cqrs.AggregateModel, error) {
	a.model = raEvents.NewResourceStateSnapshotTaken(func(deviceId, resourceId string) error {
		if a.userDeviceIds.HasOneOf(deviceId) {
			return nil
		}
		return fmt.Errorf("access denied")
	})
	return a.model, nil
}

// NewAggregate creates new resource aggreate - it must be created for every run command.
func NewAggregate(ctx context.Context, resourceId string, userDeviceIds []string, SnapshotThreshold int, eventstore EventStore, retry cqrs.RetryFunc) (*aggregate, error) {
	deviceIds := make(strings.Set)
	deviceIds.Add(userDeviceIds...)
	a := &aggregate{
		resourceId:    resourceId,
		userDeviceIds: deviceIds,
		eventstore:    eventstore,
	}
	cqrsAg, err := cqrs.NewAggregate(resourceId, retry, SnapshotThreshold, eventstore, a.factoryModel, log.Debugf)
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

	instanceID, err := a.eventstore.GetInstanceId(ctx, request.ResourceId)
	if err != nil {
		err = status.Errorf(codes.InvalidArgument, "unable to get instanceID for publish command: %v", err)
		return
	}
	request.Resource.InstanceId = instanceID

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		a.eventstore.RemoveInstanceId(ctx, instanceID)
		err = fmt.Errorf("unable to process publish command: %w", err)
		return
	}
	for _, event := range events {
		if rp, ok := event.(raEvents.ResourcePublished); ok {
			// if resource is already published we need to use origin intanceId that was set by model.
			if rp.Resource.InstanceId != instanceID {
				a.eventstore.RemoveInstanceId(ctx, instanceID)
				instanceID = rp.Resource.InstanceId
			}
			break
		}
	}
	insertMaintenanceDbRecord(ctx, a, events)

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), "")
	response = &pb.PublishResourceResponse{
		AuditContext: &auditContext,
		InstanceId:   instanceID,
	}
	return
}

// HandleUnpublishResource handles a command UnpublishResource
func (a *aggregate) UnpublishResource(ctx context.Context, request *pb.UnpublishResourceRequest) (response *pb.UnpublishResourceResponse, events []cqrsEvent.Event, err error) {
	if err = validateUnpublish(request); err != nil {
		err = fmt.Errorf("invalid unpublish command: %w", err)
		return
	}

	events, err = a.ag.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process unpublish command: %w", err)
		return
	}
	insertMaintenanceDbRecord(ctx, a, events)

	err = a.eventstore.RemoveInstanceId(ctx, a.model.Resource.InstanceId)
	if err != nil {
		err = status.Errorf(codes.Internal, "unable remove instanceID: %v", err)
	}

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), "")
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

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), "")
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

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), request.CorrelationId)
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

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), request.CorrelationId)
	response = &pb.ConfirmResourceUpdateResponse{
		AuditContext: &auditContext,
	}
	return
}

// RetrieveResource handles a command UpdateResource
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

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), request.CorrelationId)
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

	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext(), request.CorrelationId)
	response = &pb.ConfirmResourceRetrieveResponse{
		AuditContext: &auditContext,
	}
	return
}

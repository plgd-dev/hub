package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	raEvents "github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/grpc"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

type isUserDeviceFunc = func(ctx context.Context, userID, deviceID string) (bool, error)

//RequestHandler for handling incoming request
type RequestHandler struct {
	UnimplementedResourceAggregateServer
	config           Config
	eventstore       EventStore
	publisher        eventbus.Publisher
	isUserDeviceFunc isUserDeviceFunc
}

func userDevicesChanged(ctx context.Context, userID string, addedDevices, removedDevices, currentDevices map[string]bool) {
	log.Debugf("userDevicesChanged %v: added: %+v removed: %+v current: %+v\n", userID, addedDevices, removedDevices, currentDevices)
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(config Config, eventstore EventStore, publisher eventbus.Publisher, isUserDeviceFunc isUserDeviceFunc) *RequestHandler {
	return &RequestHandler{
		config:           config,
		eventstore:       eventstore,
		publisher:        publisher,
		isUserDeviceFunc: isUserDeviceFunc,
	}
}

func publishEvents(ctx context.Context, publisher eventbus.Publisher, deviceId, resourceId string, events []eventbus.Event) error {
	var errors []error
	for _, event := range events {
		err := publisher.Publish(ctx, utils.GetTopics(deviceId), deviceId, resourceId, event)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot publish events: %v", errors)
	}
	return nil
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}

func (r RequestHandler) validateAccessToDevice(ctx context.Context, deviceID string) (string, error) {
	userID, err := grpc.UserIDFromMD(ctx)
	if err != nil {
		return "", kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "invalid userID: %v", err)
	}
	ok, err := r.isUserDeviceFunc(ctx, userID, deviceID)
	if err != nil {
		return "", kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate : %v", err)
	}
	if ok {
		return userID, nil
	}
	return "", kitNetGrpc.ForwardErrorf(codes.PermissionDenied, "access denied")
}

func (r RequestHandler) PublishResourceLinks(ctx context.Context, request *commands.PublishResourceLinksRequest) (*commands.PublishResourceLinksResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	resID := commands.NewResourceID(request.DeviceId, commands.ResourceLinksHref)
	aggregate, err := NewAggregate(resID, r.config.SnapshotThreshold, r.eventstore, resourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot publish resource links: %v", err))
	}

	events, err := aggregate.PublishResourceLinks(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource links: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource links published events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return newPublishResourceLinksResponse(events, aggregate.DeviceID(), auditContext), nil
}

func newPublishResourceLinksResponse(events []eventstore.Event, deviceID string, auditContext *commands.AuditContext) *commands.PublishResourceLinksResponse {
	for _, event := range events {
		if rlp, ok := event.(*raEvents.ResourceLinksPublished); ok {
			return &commands.PublishResourceLinksResponse{
				AuditContext:       auditContext,
				PublishedResources: rlp.Resources,
				DeviceId:           deviceID,
			}
		}
	}
	return &commands.PublishResourceLinksResponse{
		AuditContext: auditContext,
		DeviceId:     deviceID,
	}
}

func (r RequestHandler) UnpublishResourceLinks(ctx context.Context, request *commands.UnpublishResourceLinksRequest) (*commands.UnpublishResourceLinksResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	resID := commands.NewResourceID(request.DeviceId, commands.ResourceLinksHref)
	aggregate, err := NewAggregate(resID, r.config.SnapshotThreshold, r.eventstore, resourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot unpublish resource links: %v", err))
	}

	events, err := aggregate.UnpublishResourceLinks(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot unpublish resource links: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource links unpublished events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return newUnpublishResourceLinksResponse(events, aggregate.DeviceID(), auditContext), nil
}

func newUnpublishResourceLinksResponse(events []eventstore.Event, deviceID string, auditContext *commands.AuditContext) *commands.UnpublishResourceLinksResponse {
	for _, event := range events {
		if rlu, ok := event.(*raEvents.ResourceLinksUnpublished); ok {
			return &commands.UnpublishResourceLinksResponse{
				AuditContext:     auditContext,
				UnpublishedHrefs: rlu.Hrefs,
				DeviceId:         deviceID,
			}
		}
	}
	return &commands.UnpublishResourceLinksResponse{
		AuditContext: auditContext,
		DeviceId:     deviceID,
	}
}

func (r RequestHandler) NotifyResourceChanged(ctx context.Context, request *commands.NotifyResourceChangedRequest) (*commands.NotifyResourceChangedResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot notify about resource content change: %v", err))
	}

	events, err := aggregate.NotifyResourceChanged(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot notify about resource content change: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource content changed notification events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return &commands.NotifyResourceChangedResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) UpdateResource(ctx context.Context, request *commands.UpdateResourceRequest) (*commands.UpdateResourceResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource content: %v", err))
	}

	events, err := aggregate.UpdateResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource content: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource content update events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.UpdateResourceResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceUpdate(ctx context.Context, request *commands.ConfirmResourceUpdateRequest) (*commands.ConfirmResourceUpdateResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource content update: %v", err))
	}

	events, err := aggregate.ConfirmResourceUpdate(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot confirm resource content update: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource content update confirmation events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.ConfirmResourceUpdateResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) RetrieveResource(ctx context.Context, request *commands.RetrieveResourceRequest) (*commands.RetrieveResourceResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot retrieve resource content: %v", err))
	}

	events, err := aggregate.RetrieveResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource content: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource content retrieve events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.RetrieveResourceResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceRetrieve(ctx context.Context, request *commands.ConfirmResourceRetrieveRequest) (*commands.ConfirmResourceRetrieveResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "ccannot confirm resource content retrieve: %v", err))
	}

	events, err := aggregate.ConfirmResourceRetrieve(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot confirm resource content retrieve: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource content retrieve confirmation events: %v", err)
	}

	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.ConfirmResourceRetrieveResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) DeleteResource(ctx context.Context, request *commands.DeleteResourceRequest) (*commands.DeleteResourceResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete resource: %v", err))
	}

	events, err := aggregate.DeleteResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish delete resource events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.DeleteResourceResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceDelete(ctx context.Context, request *commands.ConfirmResourceDeleteRequest) (*commands.ConfirmResourceDeleteResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource deletion: %v", err))
	}

	events, err := aggregate.ConfirmResourceDelete(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot confirm resource deletion: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource delete confirmation events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.ConfirmResourceDeleteResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) CreateResource(ctx context.Context, request *commands.CreateResourceRequest) (*commands.CreateResourceResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot create resource: %v", err))
	}

	events, err := aggregate.CreateResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource create events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.CreateResourceResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceCreate(ctx context.Context, request *commands.ConfirmResourceCreateRequest) (*commands.ConfirmResourceCreateResponse, error) {
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err))
	}

	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, resourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource creation: %v", err))
	}

	events, err := aggregate.ConfirmResourceCreate(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot confirm resource creation: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.ResourceID(), events)
	if err != nil {
		log.Errorf("cannot publish resource create confirmation events: %v", err)
	}
	auditContext := commands.NewAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &commands.ConfirmResourceCreateResponse{
		AuditContext: auditContext,
	}, nil
}

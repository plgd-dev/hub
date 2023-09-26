package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
)

type getOwnerDevicesFunc = func(ctx context.Context, owner string, deviceIDs []string) ([]string, error)

// RequestHandler for handling incoming request
type RequestHandler struct {
	UnimplementedResourceAggregateServer
	config              Config
	eventstore          *mongodb.EventStore
	publisher           eventbus.Publisher
	getOwnerDevicesFunc getOwnerDevicesFunc
	logger              log.Logger
	serviceStatus       *ServiceStatus
}

// NewRequestHandler factory for new RequestHandler
func NewRequestHandler(config Config, eventstore *mongodb.EventStore, publisher eventbus.Publisher, getOwnerDevicesFunc getOwnerDevicesFunc, serviceStatus *ServiceStatus, logger log.Logger) *RequestHandler {
	return &RequestHandler{
		config:              config,
		eventstore:          eventstore,
		publisher:           publisher,
		getOwnerDevicesFunc: getOwnerDevicesFunc,
		logger:              logger,
		serviceStatus:       serviceStatus,
	}
}

func PublishEvents(pub eventbus.Publisher, owner, deviceID, resourceID string, events []eventbus.Event, logger log.Logger) {
	for _, event := range events {
		// timeout si driven by flusherTimeout.
		subjects := utils.GetPublishSubject(owner, event)
		err := pub.Publish(context.Background(), subjects, deviceID, resourceID, event)
		publisher.LogPublish(logger, event, subjects, err)
	}
}

// Check if device with given ID belongs to given owner
func (r RequestHandler) isUserDevice(ctx context.Context, owner string, deviceID string) (bool, error) {
	deviceIds, err := r.getOwnerDevicesFunc(ctx, owner, []string{deviceID})
	if err != nil {
		return false, err
	}
	return len(deviceIds) == 1, nil
}

func (r RequestHandler) validateAccessToDevice(ctx context.Context, deviceID string) (string, string, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return "", "", grpc.ForwardErrorf(codes.InvalidArgument, "invalid userID: %v", err)
	}
	owner, err := grpc.OwnerFromTokenMD(ctx, r.config.APIs.GRPC.Authorization.OwnerClaim)
	if err != nil {
		return "", "", grpc.ForwardErrorf(codes.InvalidArgument, "invalid owner: %v", err)
	}
	err = r.validateAccessToDeviceWithOwner(ctx, deviceID, owner)
	if err != nil {
		return "", "", err
	}

	return userID, owner, nil
}

func (r RequestHandler) validateAccessToDeviceWithOwner(ctx context.Context, deviceID, owner string) error {
	ok, err := r.isUserDevice(ctx, owner, deviceID)
	if err != nil {
		return grpc.ForwardErrorf(codes.Internal, "cannot validate: %v", err)
	}
	if !ok {
		return grpc.ForwardErrorf(codes.PermissionDenied, "access denied")
	}
	return nil
}

// Return owner and list of owned devices from the input slices.
//
// Function iterates over input slice of device IDs and returns owner name, and the intersection
// of the input device IDs with owned devices.
func (r RequestHandler) getOwnedDevices(ctx context.Context, deviceIDs []string) (string, string, []string, error) {
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return "", "", nil, grpc.ForwardErrorf(codes.InvalidArgument, "invalid userID: %v", err)
	}

	owner, err := grpc.OwnerFromTokenMD(ctx, r.config.APIs.GRPC.Authorization.OwnerClaim)
	if err != nil {
		return "", "", nil, grpc.ForwardErrorf(codes.InvalidArgument, "invalid owner: %v", err)
	}

	ownedDevices, err := r.getOwnerDevicesFunc(ctx, owner, deviceIDs)
	if err != nil {
		return "", "", nil, grpc.ForwardErrorf(codes.InvalidArgument, "cannot validate: %v", err)
	}
	return userID, owner, ownedDevices, nil
}

func cannotValidateAccessError(err error) error {
	return grpc.ForwardErrorf(codes.Internal, "cannot validate user access: %v", err)
}

func (r RequestHandler) PublishResourceLinks(ctx context.Context, request *commands.PublishResourceLinksRequest) (*commands.PublishResourceLinksResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}

	resID := commands.NewResourceID(request.DeviceId, commands.ResourceLinksHref)
	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceLinksFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot publish resource links: %v", err))
	}

	events, err := aggregate.PublishResourceLinks(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot publish resource links: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)
	auditContext := commands.NewAuditContext(userID, "", owner)
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
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}

	resID := commands.NewResourceID(request.DeviceId, commands.ResourceLinksHref)
	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceLinksFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot unpublish resource links: %v", err))
	}

	events, err := aggregate.UnpublishResourceLinks(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot unpublish resource links: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)
	auditContext := commands.NewAuditContext(userID, "", owner)

	resp := newUnpublishResourceLinksResponse(events, aggregate.DeviceID(), auditContext)
	for _, href := range resp.GetUnpublishedHrefs() {
		_, err = r.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
			ResourceId: commands.NewResourceID(resp.GetDeviceId(), href),
			Content: &commands.Content{
				CoapContentFormat: -1,
			},
			Status:          commands.Status_NOT_FOUND,
			CommandMetadata: request.GetCommandMetadata(),
		})
		if err != nil {
			log.Errorf("cannot reset content for unpublished resource %v%v : %v", err, resp.GetDeviceId(), href)
		}
	}

	return resp, nil
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

func (r RequestHandler) notifyResourceChanged(ctx context.Context, request *commands.NotifyResourceChangedRequest, userID, owner string) error {
	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot notify about resource content change: %v", err))
	}

	events, err := aggregate.NotifyResourceChanged(ctx, request)
	if err != nil {
		return log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot notify about resource content change: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)
	return nil
}

func (r RequestHandler) NotifyResourceChanged(ctx context.Context, request *commands.NotifyResourceChangedRequest) (*commands.NotifyResourceChangedResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}

	err = r.notifyResourceChanged(ctx, request, userID, owner)
	if err != nil {
		return nil, err
	}

	auditContext := commands.NewAuditContext(userID, "", owner)
	return &commands.NotifyResourceChangedResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) UpdateResource(ctx context.Context, request *commands.UpdateResourceRequest) (*commands.UpdateResourceResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	request.TimeToLive = checkTimeToLiveForDefault(r.config.Clients.Eventstore.DefaultCommandTimeToLive, request.GetTimeToLive())

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource content: %v", err))
	}

	events, err := aggregate.UpdateResource(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot update resource content: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	var validUntil int64
	for _, e := range events {
		if ev, ok := e.(*raEvents.ResourceUpdatePending); ok {
			validUntil = ev.GetValidUntil()
			break
		}
	}

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.UpdateResourceResponse{
		AuditContext: auditContext,
		ValidUntil:   validUntil,
	}, nil
}

func (r RequestHandler) ConfirmResourceUpdate(ctx context.Context, request *commands.ConfirmResourceUpdateRequest) (*commands.ConfirmResourceUpdateResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource content update: %v", err))
	}

	events, err := aggregate.ConfirmResourceUpdate(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot confirm resource content update: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)
	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.ConfirmResourceUpdateResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) RetrieveResource(ctx context.Context, request *commands.RetrieveResourceRequest) (*commands.RetrieveResourceResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	request.TimeToLive = checkTimeToLiveForDefault(r.config.Clients.Eventstore.DefaultCommandTimeToLive, request.GetTimeToLive())

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot retrieve resource content: %v", err))
	}

	events, err := aggregate.RetrieveResource(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot retrieve resource content: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	var validUntil int64
	for _, e := range events {
		if ev, ok := e.(*raEvents.ResourceRetrievePending); ok {
			validUntil = ev.GetValidUntil()
			break
		}
	}

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.RetrieveResourceResponse{
		AuditContext: auditContext,
		ValidUntil:   validUntil,
	}, nil
}

func (r RequestHandler) ConfirmResourceRetrieve(ctx context.Context, request *commands.ConfirmResourceRetrieveRequest) (*commands.ConfirmResourceRetrieveResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "ccannot confirm resource content retrieve: %v", err))
	}

	events, err := aggregate.ConfirmResourceRetrieve(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot confirm resource content retrieve: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.ConfirmResourceRetrieveResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) DeleteResource(ctx context.Context, request *commands.DeleteResourceRequest) (*commands.DeleteResourceResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	request.TimeToLive = checkTimeToLiveForDefault(r.config.Clients.Eventstore.DefaultCommandTimeToLive, request.GetTimeToLive())

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot delete resource: %v", err))
	}

	events, err := aggregate.DeleteResource(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	var validUntil int64
	for _, e := range events {
		if ev, ok := e.(*raEvents.ResourceDeletePending); ok {
			validUntil = ev.GetValidUntil()
			break
		}
	}

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.DeleteResourceResponse{
		AuditContext: auditContext,
		ValidUntil:   validUntil,
	}, nil
}

func (r RequestHandler) ConfirmResourceDelete(ctx context.Context, request *commands.ConfirmResourceDeleteRequest) (*commands.ConfirmResourceDeleteResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource deletion: %v", err))
	}

	events, err := aggregate.ConfirmResourceDelete(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot confirm resource deletion: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.ConfirmResourceDeleteResponse{
		AuditContext: auditContext,
	}, nil
}

func (r RequestHandler) CreateResource(ctx context.Context, request *commands.CreateResourceRequest) (*commands.CreateResourceResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}
	request.TimeToLive = checkTimeToLiveForDefault(r.config.Clients.Eventstore.DefaultCommandTimeToLive, request.GetTimeToLive())

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot create resource: %v", err))
	}

	events, err := aggregate.CreateResource(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	var validUntil int64
	for _, e := range events {
		if ev, ok := e.(*raEvents.ResourceCreatePending); ok {
			validUntil = ev.GetValidUntil()
			break
		}
	}

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.CreateResourceResponse{
		AuditContext: auditContext,
		ValidUntil:   validUntil,
	}, nil
}

func (r RequestHandler) ConfirmResourceCreate(ctx context.Context, request *commands.ConfirmResourceCreateRequest) (*commands.ConfirmResourceCreateResponse, error) {
	userID, owner, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, log.LogAndReturnError(cannotValidateAccessError(err))
	}

	aggregate, err := NewAggregate(request.ResourceId, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewResourceStateFactoryModel(userID, owner, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot confirm resource creation: %v", err))
	}

	events, err := aggregate.ConfirmResourceCreate(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(grpc.ForwardErrorf(codes.Internal, "cannot confirm resource creation: %v", err))
	}

	PublishEvents(r.publisher, owner, aggregate.DeviceID(), aggregate.ResourceID(), events, r.logger)

	auditContext := commands.NewAuditContext(userID, request.GetCorrelationId(), owner)
	return &commands.ConfirmResourceCreateResponse{
		AuditContext: auditContext,
	}, nil
}

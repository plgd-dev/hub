package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"

	cqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	cqrs "github.com/plgd-dev/cqrs"
	cqrsEvent "github.com/plgd-dev/cqrs/event"
	cqrsEventBus "github.com/plgd-dev/cqrs/eventbus"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/grpc"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

type isUserDeviceFunc = func(ctx context.Context, userID, deviceID string) (bool, error)

//RequestHandler for handling incoming request
type RequestHandler struct {
	pb.UnimplementedResourceAggregateServer
	config           Config
	eventstore       EventStore
	publisher        cqrsEventBus.Publisher
	isUserDeviceFunc isUserDeviceFunc
}

func userDevicesChanged(ctx context.Context, userID string, addedDevices, removedDevices, currentDevices map[string]bool) {
	log.Debugf("userDevicesChanged %v: added: %+v removed: %+v current: %+v\n", userID, addedDevices, removedDevices, currentDevices)
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(config Config, eventstore EventStore, publisher cqrsEventBus.Publisher, isUserDeviceFunc isUserDeviceFunc) *RequestHandler {
	return &RequestHandler{
		config:           config,
		eventstore:       eventstore,
		publisher:        publisher,
		isUserDeviceFunc: isUserDeviceFunc,
	}
}

func publishEvents(ctx context.Context, publisher cqrsEventBus.Publisher, deviceId, resourceId string, events []cqrsEvent.Event) error {
	var errors []error
	for _, event := range events {
		err := publisher.Publish(ctx, cqrsUtils.GetTopics(deviceId), deviceId, resourceId, event)
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

func (r RequestHandler) PublishResource(ctx context.Context, request *pb.PublishResourceRequest) (*pb.PublishResourceResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.PublishResource(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}

	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot publish resource: %v", err))
	}

	events, err := aggregate.PublishResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for publish command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return &pb.PublishResourceResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) UnpublishResource(ctx context.Context, request *pb.UnpublishResourceRequest) (*pb.UnpublishResourceResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.UnpublishResource(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot unpublish resource: %v", err))
	}

	events, err := aggregate.UnpublishResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot unpublish resource: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for unpublish command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return &pb.UnpublishResourceResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) NotifyResourceChanged(ctx context.Context, request *pb.NotifyResourceChangedRequest) (*pb.NotifyResourceChangedResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.NotifyResourceChanged(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot notify resource content changed: %v", err))
	}

	events, err := aggregate.NotifyResourceChanged(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot notify resource content changed: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for notify content changed command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, "")
	return &pb.NotifyResourceChangedResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) UpdateResource(ctx context.Context, request *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.UpdateResource(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource content: %v", err))
	}

	events, err := aggregate.UpdateResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource content: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for update resource content command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.UpdateResourceResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceUpdate(ctx context.Context, request *pb.ConfirmResourceUpdateRequest) (*pb.ConfirmResourceUpdateResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.ConfirmResourceUpdate(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot notify resource content update processed: %v", err))
	}

	events, err := aggregate.ConfirmResourceUpdate(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot notify resource content update processed: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for notify resource content update processed command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.ConfirmResourceUpdateResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) RetrieveResource(ctx context.Context, request *pb.RetrieveResourceRequest) (*pb.RetrieveResourceResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.RetrieveResource(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot retrieve resource content: %v", err))
	}

	events, err := aggregate.RetrieveResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource content: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for retrieve resource content command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.RetrieveResourceResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceRetrieve(ctx context.Context, request *pb.ConfirmResourceRetrieveRequest) (*pb.ConfirmResourceRetrieveResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.ConfirmResourceRetrieve(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot notify resource content retrieve processed: %v", err))
	}

	events, err := aggregate.ConfirmResourceRetrieve(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot notify resource content retrieve processed: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for notify resource content retrieve processed command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.ConfirmResourceRetrieveResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) DeleteResource(ctx context.Context, request *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.DeleteResource(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete resource: %v", err))
	}

	events, err := aggregate.DeleteResource(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for delete resource command: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.DeleteResourceResponse{
		AuditContext: &auditContext,
	}, nil
}

func (r RequestHandler) ConfirmResourceDelete(ctx context.Context, request *pb.ConfirmResourceDeleteRequest) (*pb.ConfirmResourceDeleteResponse, error) {
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.ConfirmResourceDelete(%v) takes %v\n", request.ResourceId, time.Now().Sub(t))
	}()
	userID, err := r.validateAccessToDevice(ctx, request.GetResourceId().GetDeviceId())
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot publish resource: %v", err))
	}
	aggregate, err := NewAggregate(request.ResourceId, r.config.SnapshotThreshold, r.eventstore, cqrs.NewDefaultRetryFunc(r.config.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot notify resource delete processed: %v", err))
	}

	events, err := aggregate.ConfirmResourceDelete(ctx, request)
	if err != nil {
		return nil, logAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot notify resource delete processed: %v", err))
	}

	err = publishEvents(ctx, r.publisher, aggregate.DeviceID(), aggregate.resourceID, events)
	if err != nil {
		log.Errorf("cannot publish events for notify resource delete processed: %v", err)
	}
	auditContext := cqrsUtils.MakeAuditContext(request.GetAuthorizationContext().GetDeviceId(), userID, request.GetCorrelationId())
	return &pb.ConfirmResourceDeleteResponse{
		AuditContext: &auditContext,
	}, nil
}

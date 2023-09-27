package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ServiceUserID               = "x.plgd.dev.service.user"
	errFmtUpdateServiceMetadata = "cannot update service metadata: %w"
)

func validateUpdateServiceMetadata(request *commands.UpdateServiceMetadataRequest) error {
	if request.GetUpdate() == nil {
		return status.Errorf(codes.InvalidArgument, "invalid update")
	}
	if request.GetStatus() == nil {
		return status.Errorf(codes.InvalidArgument, "unexpected update %T", request.GetUpdate())
	}
	if request.GetStatus().GetId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid status.id")
	}
	if request.GetStatus().GetTimeToLive() < int64(time.Second) {
		return status.Errorf(codes.InvalidArgument, "invalid status.timeToLive(%v): is less than 1s", time.Duration(request.GetStatus().GetTimeToLive()))
	}
	return nil
}

func (a *Aggregate) UpdateServiceMetadata(ctx context.Context, request *commands.UpdateServiceMetadataRequest) (events []eventstore.Event, err error) {
	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update service metadata command command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)
	return
}

func (a *Aggregate) ConfirmOfflineServices(ctx context.Context, request *events.ConfirmOfflineServicesRequest) (events []eventstore.Event, err error) {
	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update service metadata command command: %w", err)
		return
	}
	cleanUpToSnapshot(ctx, a, events)
	return
}

func (r RequestHandler) UpdateServiceMetadata(ctx context.Context, request *commands.UpdateServiceMetadataRequest) (*commands.UpdateServiceMetadataResponse, error) {
	if err := validateUpdateServiceMetadata(request); err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, errFmtUpdateServiceMetadata, err))
	}
	respChan := make(chan UpdateServiceMetadataResponseChanData, 1)
	if err := r.serviceStatus.ProcessRequest(UpdateServiceMetadataReqResp{
		Request:      request,
		ResponseChan: respChan,
	}); err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, errFmtUpdateServiceMetadata, err))
	}
	select {
	case <-ctx.Done():
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Canceled, errFmtUpdateServiceMetadata, ctx.Err()))
	case resp := <-respChan:
		if resp.Err != nil {
			return nil, resp.Err
		}
		return resp.Response, nil
	}
}

type UpdateServiceMetadataReqResp struct {
	Request      *commands.UpdateServiceMetadataRequest
	ResponseChan chan UpdateServiceMetadataResponseChanData
}

type ServiceStatus struct {
	logger     log.Logger
	config     Config
	eventstore *mongodb.EventStore
	publisher  eventbus.Publisher
	wake       chan struct{}
	closed     atomic.Bool
	wg         sync.WaitGroup

	private struct {
		mutex        sync.Mutex
		requestQueue []UpdateServiceMetadataReqResp
	}
}

func NewServiceStatus(config Config, eventstore *mongodb.EventStore, publisher eventbus.Publisher, logger log.Logger) *ServiceStatus {
	s := &ServiceStatus{
		logger:     logger,
		config:     config,
		eventstore: eventstore,
		publisher:  publisher,
		wake:       make(chan struct{}, 1),
		private: struct {
			mutex        sync.Mutex
			requestQueue []UpdateServiceMetadataReqResp
		}{
			requestQueue: make([]UpdateServiceMetadataReqResp, 0, 8),
		},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run()
	}()
	return s
}

func (s *ServiceStatus) handleOfflineService(ctx context.Context, aggregate *Aggregate, service *events.ServicesStatus_Status) error {
	const limitNumDevices = 256
	for {
		devices, err := s.eventstore.LoadDeviceMetadataByServiceIDs(ctx, []string{service.GetId()}, limitNumDevices)
		if err != nil {
			return fmt.Errorf("cannot load devices for offline services %v: %w", service.GetId(), err)
		}
		if len(devices) == 0 {
			_, err := aggregate.ConfirmOfflineServices(ctx, &events.ConfirmOfflineServicesRequest{
				Status: []*events.ServicesStatus_Status{service},
			})
			if err != nil {
				return fmt.Errorf("cannot confirm offline services metadata: %w", err)
			}
			return nil
		}
		for _, d := range devices {
			err := s.updateDeviceToOffline(ctx, d.ServiceID, d.DeviceID, ServiceUserID)
			if err != nil {
				return fmt.Errorf("cannot update device %v to offline because service %v is offline: %w", d.DeviceID, d.ServiceID, err)
			}
		}
	}
}

type UpdateServiceMetadataResponseChanData struct {
	Response *commands.UpdateServiceMetadataResponse
	Err      error
}

func (s *ServiceStatus) updateServiceMetadata(aggregate *Aggregate, r UpdateServiceMetadataReqResp) []*events.ServicesStatus_Status {
	publishEvents, err := aggregate.UpdateServiceMetadata(context.Background(), r.Request)
	if err != nil {
		select {
		case r.ResponseChan <- UpdateServiceMetadataResponseChanData{
			Err: log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, errFmtUpdateServiceMetadata, err)),
		}: // sent
		default:
		}
		return nil
	}

	var onlineValidUntil int64
	var offlineServices []*events.ServicesStatus_Status
	for _, e := range publishEvents {
		if ev, ok := e.(*events.ServicesMetadataUpdated); ok {
			for _, s := range ev.GetStatus().GetOnline() {
				if s.GetId() == r.Request.GetStatus().GetId() {
					onlineValidUntil = s.GetOnlineValidUntil()
					break
				}
			}
			offlineServices = ev.GetStatus().GetOffline()
		}
	}
	select {
	case r.ResponseChan <- UpdateServiceMetadataResponseChanData{
		Response: &commands.UpdateServiceMetadataResponse{
			OnlineValidUntil: onlineValidUntil,
		},
	}: // sent
	default:
	}
	return offlineServices
}

func (s *ServiceStatus) processRequest(r UpdateServiceMetadataReqResp) {
	ctx := context.Background()
	resID := commands.NewResourceID(s.config.HubID, commands.ServicesResourceHref)
	aggregate, err := NewAggregate(resID, s.config.Clients.Eventstore.SnapshotThreshold, s.eventstore, NewServicesMetadataFactoryModel(ServiceUserID, ServiceUserID, s.config.HubID), cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		s.logger.Errorf(errFmtUpdateServiceMetadata, err)
		return
	}
	offlineServices := s.updateServiceMetadata(aggregate, r)
	for _, off := range offlineServices {
		err := s.handleOfflineService(ctx, aggregate, off)
		if err != nil {
			s.logger.Errorf("cannot handle offline service %v: %w", off.GetId(), err)
		} else {
			s.logger.Infof("all devices associated with offline service %v have been marked as offline", off.GetId())
		}
	}
}

func (s *ServiceStatus) processRequests() {
	for {
		reqs := s.pop()
		if len(reqs) == 0 {
			return
		}
		for _, r := range reqs {
			s.processRequest(r)
		}
	}
}

func (s *ServiceStatus) run() {
	for {
		if s.closed.Load() {
			return
		}
		<-s.wake
		s.processRequests()
	}
}

func (s *ServiceStatus) push(req UpdateServiceMetadataReqResp) {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	s.private.requestQueue = append(s.private.requestQueue, req)
}

func (s *ServiceStatus) pop() []UpdateServiceMetadataReqResp {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	reqs := s.private.requestQueue
	s.private.requestQueue = make([]UpdateServiceMetadataReqResp, 0, 8)
	return reqs
}

func (s *ServiceStatus) ProcessRequest(r UpdateServiceMetadataReqResp) error {
	if r.Request == nil {
		return fmt.Errorf("invalid request")
	}
	if r.ResponseChan == nil {
		return fmt.Errorf("invalid response channel")
	}
	s.push(r)
	s.wakeUp()
	return nil
}

func (s *ServiceStatus) wakeUp() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *ServiceStatus) Close() {
	if s.closed.CompareAndSwap(false, true) {
		s.wakeUp()
		s.wg.Wait()
	}
}

func (s *ServiceStatus) updateDeviceToOffline(ctx context.Context, serviceID, deviceID, userID string) error {
	resID := commands.NewResourceID(deviceID, commands.StatusHref)

	var latestSnapshot *events.DeviceMetadataSnapshotTakenForCommand
	deviceMetadataFactoryModel := func(ctx context.Context) (cqrsAggregate.AggregateModel, error) {
		latestSnapshot = events.NewDeviceMetadataSnapshotTakenForCommand(userID, "", s.config.HubID)
		return latestSnapshot, nil
	}

	aggregate, err := NewAggregate(resID, s.config.Clients.Eventstore.SnapshotThreshold, s.eventstore, deviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot set device('%v') to offline state: %v", deviceID, err))
	}

	publishEvents, err := aggregate.UpdateDeviceToOffline(ctx, &events.UpdateDeviceToOfflineRequest{
		DeviceID:  deviceID,
		ServiceID: serviceID,
	})
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot set device('%v') to offline state: %v", deviceID, err))
	}

	PublishEvents(s.publisher, latestSnapshot.GetDeviceMetadataUpdated().GetAuditContext().GetOwner(), aggregate.DeviceID(), aggregate.ResourceID(), publishEvents, s.logger)
	return nil
}

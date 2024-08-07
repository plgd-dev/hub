package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"go.uber.org/atomic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ServiceUserID = uuid.NullUUID{Valid: true}.UUID.String()

const errFmtUpdateServiceMetadata = "cannot update service metadata: %w"

func validateUpdateServiceMetadata(request *commands.UpdateServiceMetadataRequest) error {
	if request.GetUpdate() == nil {
		return status.Errorf(codes.InvalidArgument, "invalid update")
	}
	if request.GetHeartbeat() == nil {
		return status.Errorf(codes.InvalidArgument, "unexpected update %T", request.GetUpdate())
	}
	if request.GetHeartbeat().GetServiceId() == "" {
		return status.Errorf(codes.InvalidArgument, "invalid heartbeat.serviceId")
	}
	if request.GetHeartbeat().GetTimeToLive() < int64(time.Second) {
		return status.Errorf(codes.InvalidArgument, "invalid heartbeat.timeToLive(%v): is less than 1s", time.Duration(request.GetHeartbeat().GetTimeToLive()))
	}
	return nil
}

func (a *Aggregate) UpdateServiceMetadata(ctx context.Context, request *commands.UpdateServiceMetadataRequest) (events []eventstore.Event, err error) {
	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update service metadata command command: %w", err)
		return
	}
	return
}

func (a *Aggregate) ConfirmExpiredServices(ctx context.Context, request *events.ConfirmExpiredServicesRequest) (events []eventstore.Event, err error) {
	events, err = a.HandleCommand(ctx, request)
	if err != nil {
		err = fmt.Errorf("unable to process update service metadata command command: %w", err)
		return
	}
	return
}

func (r RequestHandler) UpdateServiceMetadata(ctx context.Context, request *commands.UpdateServiceMetadataRequest) (*commands.UpdateServiceMetadataResponse, error) {
	if err := validateUpdateServiceMetadata(request); err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, errFmtUpdateServiceMetadata, err))
	}
	respChan := make(chan UpdateServiceMetadataResponseChanData, 1)
	if err := r.serviceHeartbeat.ProcessRequest(UpdateServiceMetadataReqResp{
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

type ServiceHeartbeat struct {
	logger      log.Logger
	config      Config
	eventstore  eventstore.EventStore
	publisher   eventbus.Publisher
	wake        chan struct{}
	wakeExpired chan struct{}
	closed      atomic.Bool
	wg          sync.WaitGroup

	offlineServices *coapSync.Map[string, *events.ServicesHeartbeat_Heartbeat]

	private struct {
		mutex        sync.Mutex
		requestQueue []UpdateServiceMetadataReqResp
	}
}

func NewServiceHeartbeat(config Config, eventstore eventstore.EventStore, publisher eventbus.Publisher, logger log.Logger) *ServiceHeartbeat {
	s := &ServiceHeartbeat{
		logger:          logger,
		config:          config,
		eventstore:      eventstore,
		publisher:       publisher,
		wake:            make(chan struct{}, 1),
		wakeExpired:     make(chan struct{}, 1),
		offlineServices: coapSync.NewMap[string, *events.ServicesHeartbeat_Heartbeat](),
		private: struct {
			mutex        sync.Mutex
			requestQueue []UpdateServiceMetadataReqResp
		}{
			requestQueue: make([]UpdateServiceMetadataReqResp, 0, 8),
		},
	}
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.run()
	}()
	go func() {
		defer s.wg.Done()
		s.runProcessExpiredServices()
	}()
	return s
}

func (s *ServiceHeartbeat) handleExpiredService(ctx context.Context, aggregate *Aggregate, service *events.ServicesHeartbeat_Heartbeat) error {
	const limitNumDevices = 256
	for {
		devices, err := s.eventstore.LoadDeviceMetadataByServiceIDs(ctx, []string{service.GetServiceId()}, limitNumDevices)
		if err != nil {
			return fmt.Errorf("cannot load devices for expired services %v: %w", service.GetServiceId(), err)
		}
		if len(devices) == 0 {
			_, err := aggregate.ConfirmExpiredServices(ctx, &events.ConfirmExpiredServicesRequest{
				Heartbeat: []*events.ServicesHeartbeat_Heartbeat{service},
			})
			if err != nil {
				return fmt.Errorf("cannot confirm expired services metadata: %w", err)
			}
			return nil
		}
		for _, d := range devices {
			err := s.updateDeviceToExpired(ctx, d.ServiceID, d.DeviceID, ServiceUserID)
			if err != nil {
				return fmt.Errorf("cannot update device %v to expired because service %v is expired: %w", d.DeviceID, d.ServiceID, err)
			}
		}
	}
}

type UpdateServiceMetadataResponseChanData struct {
	Response *commands.UpdateServiceMetadataResponse
	Err      error
}

func (s *ServiceHeartbeat) updateServiceMetadata(aggregate *Aggregate, r UpdateServiceMetadataReqResp) {
	publishEvents, err := aggregate.UpdateServiceMetadata(context.Background(), r.Request)
	if err != nil {
		select {
		case r.ResponseChan <- UpdateServiceMetadataResponseChanData{
			Err: log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, errFmtUpdateServiceMetadata, err)),
		}: // sent
		default:
		}
	}

	var heartbeatValidUntil int64
	for _, e := range publishEvents {
		if ev, ok := e.(*events.ServiceMetadataUpdated); ok {
			for _, s := range ev.GetServicesHeartbeat().GetValid() {
				if s.GetServiceId() == r.Request.GetHeartbeat().GetServiceId() {
					heartbeatValidUntil = s.GetValidUntil()
					break
				}
			}
		}
	}
	select {
	case r.ResponseChan <- UpdateServiceMetadataResponseChanData{
		Response: &commands.UpdateServiceMetadataResponse{
			HeartbeatValidUntil: heartbeatValidUntil,
		},
	}: // sent
	default:
	}
}

func (s *ServiceHeartbeat) processRequest(r UpdateServiceMetadataReqResp) time.Time {
	resID := commands.NewResourceID(s.config.HubID, commands.ServicesResourceHref)
	var snapshot *events.ServiceMetadataSnapshotTakenForCommand
	newServicesMetadataFactoryModelFunc := func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		snapshot = events.NewServiceMetadataSnapshotTakenForCommand(ServiceUserID, ServiceUserID, s.config.HubID)
		return snapshot, nil
	}
	nextWakeUp := pkgTime.MaxTime
	aggregate, err := NewAggregate(resID, s.eventstore, newServicesMetadataFactoryModelFunc, cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		s.logger.Errorf(errFmtUpdateServiceMetadata, err)
		return nextWakeUp
	}
	s.updateServiceMetadata(aggregate, r)

	for _, expired := range snapshot.GetServiceMetadataUpdated().GetServicesHeartbeat().GetExpired() {
		s.offlineServices.Store(expired.GetServiceId(), expired)
	}
	for _, valid := range snapshot.GetServiceMetadataUpdated().GetServicesHeartbeat().GetValid() {
		validUntil := pkgTime.Unix(0, valid.GetValidUntil())
		if validUntil.Before(nextWakeUp) {
			nextWakeUp = time.Unix(0, valid.GetValidUntil())
		}
	}
	if s.offlineServices.Length() > 0 {
		s.wakeUpProcessingExpired()
	}
	return nextWakeUp
}

func (s *ServiceHeartbeat) processRequests() time.Time {
	nextWakeUp := pkgTime.MaxTime
	for {
		reqs := s.pop()
		if len(reqs) == 0 {
			return nextWakeUp
		}
		for _, r := range reqs {
			if t := s.processRequest(r); t.Before(nextWakeUp) {
				nextWakeUp = t
			}
		}
	}
}

func (s *ServiceHeartbeat) processExpiredServices() {
	offlineServices := s.offlineServices.CopyData()
	ctx := context.Background()
	resID := commands.NewResourceID(s.config.HubID, commands.ServicesResourceHref)
	aggregate, err := NewAggregate(resID, s.eventstore, NewServicesMetadataFactoryModel(ServiceUserID, ServiceUserID, s.config.HubID), cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		s.logger.Errorf(errFmtUpdateServiceMetadata, err)
		return
	}
	for _, off := range offlineServices {
		err := s.handleExpiredService(ctx, aggregate, off)
		if err != nil {
			s.logger.Errorf("cannot handle expired service %v: %w", off.GetServiceId(), err)
		} else {
			s.logger.Infof("all devices associated with expired service %v have been marked as offline", off.GetServiceId())
			s.offlineServices.Delete(off.GetServiceId())
		}
	}
}

func (s *ServiceHeartbeat) runProcessExpiredServices() {
	for {
		if s.closed.Load() {
			return
		}
		<-s.wakeExpired
		s.processExpiredServices()
	}
}

func (s *ServiceHeartbeat) pushCheckExpiredServices() {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()

	for _, req := range s.private.requestQueue {
		// check if request is already in queue
		if req.Request.GetHeartbeat() != nil && req.Request.GetHeartbeat().GetServiceId() == "" && req.Request.GetHeartbeat().GetTimeToLive() == 0 {
			return
		}
	}

	s.private.requestQueue = append(s.private.requestQueue, UpdateServiceMetadataReqResp{
		Request: &commands.UpdateServiceMetadataRequest{
			Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
				Heartbeat: &commands.ServiceHeartbeat{},
			},
		},
		ResponseChan: make(chan UpdateServiceMetadataResponseChanData, 1),
	})
}

func (s *ServiceHeartbeat) run() {
	// wake up immediately to check expired services
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		if s.closed.Load() {
			return
		}
		select {
		case <-s.wake:
		case <-timer.C:
			// push heartbeat to check expired services
			s.pushCheckExpiredServices()
		}
		nextWakeUp := s.processRequests()
		d := time.Until(nextWakeUp)
		if d <= 0 {
			d = time.Minute * 10
		}
		// stop timer and drain if it was not already expired
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(d)
	}
}

func (s *ServiceHeartbeat) push(req UpdateServiceMetadataReqResp) {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	s.private.requestQueue = append(s.private.requestQueue, req)
}

func (s *ServiceHeartbeat) pop() []UpdateServiceMetadataReqResp {
	s.private.mutex.Lock()
	defer s.private.mutex.Unlock()
	reqs := s.private.requestQueue
	s.private.requestQueue = make([]UpdateServiceMetadataReqResp, 0, 8)
	return reqs
}

func (s *ServiceHeartbeat) ProcessRequest(r UpdateServiceMetadataReqResp) error {
	if r.Request == nil {
		return errors.New("invalid request")
	}
	if r.ResponseChan == nil {
		return errors.New("invalid response channel")
	}
	s.push(r)
	s.wakeUp()
	return nil
}

func (s *ServiceHeartbeat) wakeUpProcessingExpired() {
	select {
	case s.wakeExpired <- struct{}{}:
	default:
	}
}

func (s *ServiceHeartbeat) wakeUp() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *ServiceHeartbeat) Close() {
	if s.closed.CompareAndSwap(false, true) {
		s.wakeUp()
		s.wakeUpProcessingExpired()
		s.wg.Wait()
	}
}

func (s *ServiceHeartbeat) updateDeviceToExpired(ctx context.Context, serviceID, deviceID, userID string) error {
	resID := commands.NewResourceID(deviceID, commands.StatusHref)

	var latestSnapshot *events.DeviceMetadataSnapshotTakenForCommand
	deviceMetadataFactoryModel := func(context.Context, string, string) (cqrsAggregate.AggregateModel, error) {
		latestSnapshot = events.NewDeviceMetadataSnapshotTakenForCommand(userID, "", s.config.HubID)
		return latestSnapshot, nil
	}

	aggregate, err := NewAggregate(resID, s.eventstore, deviceMetadataFactoryModel, cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
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

	// TODO: In the future, we need to retrieve the owner from the identity-store service.
	PublishEvents(s.publisher, latestSnapshot.GetDeviceMetadataUpdated().GetAuditContext().GetOwner(), aggregate.DeviceID(), aggregate.ResourceID(), publishEvents, s.logger)
	return nil
}

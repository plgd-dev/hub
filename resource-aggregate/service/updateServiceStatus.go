package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ServiceUserID = "x.plgd.dev.service.user"

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
	if err = validateUpdateServiceMetadata(request); err != nil {
		err = fmt.Errorf("invalid update service metadata command: %w", err)
		return
	}

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

func mapServiceStatusesToArray(services map[string]*events.ServicesStatus_Status) []*events.ServicesStatus_Status {
	arr := make([]*events.ServicesStatus_Status, 0, len(services))
	for _, s := range services {
		arr = append(arr, s)
	}
	return arr
}

func (r RequestHandler) UpdateServiceMetadata(ctx context.Context, request *commands.UpdateServiceMetadataRequest) (*commands.UpdateServiceMetadataResponse, error) {
	resID := commands.NewResourceID(r.config.HubID, commands.ServicesResourceHref)

	aggregate, err := NewAggregate(resID, r.config.Clients.Eventstore.SnapshotThreshold, r.eventstore, NewServicesMetadataFactoryModel(ServiceUserID, ServiceUserID, r.config.HubID), cqrsAggregate.NewDefaultRetryFunc(r.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update services metadata: %v", err))
	}

	publishEvents, err := aggregate.UpdateServiceMetadata(ctx, request)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update services metadata: %v", err))
	}

	for _, e := range publishEvents {
		if ev, ok := e.(*events.ServicesMetadataUpdated); ok {
			r.serviceStatus.SetOfflineServices(ev.GetStatus().GetOffline())
		}
	}
	return &commands.UpdateServiceMetadataResponse{}, nil
}

type ServiceStatus struct {
	queue           *queue.Queue
	offlineServices *sync.Map[string, *events.ServicesStatus_Status]
	logger          log.Logger
	config          Config
	eventstore      *mongodb.EventStore
	publisher       eventbus.Publisher
}

func NewServiceStatus(config Config, eventstore *mongodb.EventStore, publisher eventbus.Publisher, logger log.Logger) (*ServiceStatus, error) {
	queue, err := queue.New(queue.Config{
		GoPoolSize:  1,
		Size:        1,
		MaxIdleTime: time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create queue: %w", err)
	}
	return &ServiceStatus{
		queue:           queue,
		offlineServices: sync.NewMap[string, *events.ServicesStatus_Status](),
		logger:          logger,
		config:          config,
		eventstore:      eventstore,
		publisher:       publisher,
	}, nil
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

func (s *ServiceStatus) task() {
	resID := commands.NewResourceID(s.config.HubID, commands.ServicesResourceHref)
	offlineServices := mapServiceStatusesToArray(s.offlineServices.CopyData())
	ctx := context.Background()
	aggregate, err := NewAggregate(resID, s.config.Clients.Eventstore.SnapshotThreshold, s.eventstore, NewServicesMetadataFactoryModel(ServiceUserID, ServiceUserID, s.config.HubID), cqrsAggregate.NewDefaultRetryFunc(s.config.Clients.Eventstore.ConcurrencyExceptionMaxRetry))
	if err != nil {
		s.logger.Errorf("cannot update services metadata: %v", err)
	}
	for _, off := range offlineServices {
		err := s.handleOfflineService(ctx, aggregate, off)
		if err != nil {
			s.logger.Errorf("cannot handle offline service %v: %v", off.GetId(), err)
		} else {
			s.offlineServices.Delete(off.GetId())
			s.logger.Infof("all devices associated with offline service %v have been marked as offline", off.GetId())
		}
	}
}

func (s *ServiceStatus) SetOfflineServices(offline []*events.ServicesStatus_Status) {
	if len(offline) == 0 {
		return
	}
	for _, o := range offline {
		s.offlineServices.Store(o.GetId(), o)
	}
	// if queue is full, it will return error. But it means that another task is already planned.
	_ = s.queue.Submit(s.task)
}

func (s *ServiceStatus) NumOfflineServices() int {
	return s.offlineServices.Length()
}

func (s *ServiceStatus) Close() {
	s.queue.Release()
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

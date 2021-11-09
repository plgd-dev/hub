package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-gateway/store"
	"github.com/plgd-dev/hub/grpc-gateway/client"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Subscription interface {
	Cancel() (func(), error)
}

type SubscriptionData struct {
	incrementSubscriptionSequenceNumber func(ctx context.Context, subscriptionID string) (uint64, error)
	setInitialized                      func(ctx context.Context, subscriptionID string) error
	gwClient                            pb.GrpcGatewayClient

	mutex sync.Mutex
	sub   Subscription
	data  store.Subscription
}

func (s *SubscriptionData) Subscription() Subscription {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.sub
}

func (s *SubscriptionData) Data() store.Subscription {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.data
}

func (s *SubscriptionData) Store(sub Subscription) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sub = sub
}

func (s *SubscriptionData) detectDevicesState(ctx context.Context) (hasDevice, hasOnline, hasOffline bool, err error) {
	client, err := s.gwClient.GetDevicesMetadata(ctx, &pb.GetDevicesMetadataRequest{})
	if err != nil {
		return false, false, false, err
	}
	for {
		d, err := client.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			break
		}
		if err != nil {
			return false, false, false, err
		}
		hasDevice = true
		if d.GetStatus().IsOnline() {
			hasOnline = true
		} else {
			hasOffline = true
		}
	}
	return hasDevice, hasOnline, hasOffline, nil
}

func (s *SubscriptionData) getEventsToEmitForDevicesSubscription(ctx context.Context) ([]events.EventType, error) {
	hasDev, hasOnlineDev, hasOfflineDev, err := s.detectDevicesState(ctx)
	if err != nil {
		return nil, err
	}
	var eventTypes []events.EventType
	if s.data.EventTypes.Has(events.EventType_DevicesRegistered) && !hasDev {
		eventTypes = append(eventTypes, events.EventType_DevicesRegistered)
	}
	if s.data.EventTypes.Has(events.EventType_DevicesUnregistered) {
		eventTypes = append(eventTypes, events.EventType_DevicesUnregistered)
	}
	if s.data.EventTypes.Has(events.EventType_DevicesOnline) && !hasOnlineDev {
		eventTypes = append(eventTypes, events.EventType_DevicesOnline)
	}
	if s.data.EventTypes.Has(events.EventType_DevicesOffline) && !hasOfflineDev {
		eventTypes = append(eventTypes, events.EventType_DevicesOffline)
	}
	return eventTypes, nil
}

func (s *SubscriptionData) getDevicesOnlineOfflineEventHandler(emitEvent emitEventFunc) interface{} {
	devsHandler := devicesSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}
	switch {
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devsHandler
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		return &devicesRegisteredOnlineOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devicesUnregisteredOnlineOfflineHandler{
			h: &devsHandler,
		}
	}

	return &devicesOnlineOfflineHandler{
		h: &devsHandler,
	}
}

func (s *SubscriptionData) getDevicesOnlineEventHandler(emitEvent emitEventFunc) interface{} {
	devsHandler := devicesSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}

	switch {
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devicesRegisteredUnregisteredOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		return &devicesRegisteredOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devicesUnregisteredOnlineHandler{
			h: &devsHandler,
		}
	}

	return &devicesOnlineHandler{
		h: &devsHandler,
	}
}

func (s *SubscriptionData) getDevicesOfflineEventHandler(emitEvent emitEventFunc) interface{} {
	devsHandler := devicesSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}

	switch {
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devicesRegisteredUnregisteredOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		return &devicesRegisteredOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		return &devicesUnregisteredOfflineHandler{
			h: &devsHandler,
		}
	}

	return &devicesOfflineHandler{
		h: &devsHandler,
	}
}

func (s *SubscriptionData) getEventHandlerForDevicesSubscription(emitEvent emitEventFunc) (interface{}, error) {
	devsHandler := devicesSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}

	isOnlineEvent := s.data.EventTypes.Has(events.EventType_DevicesOnline)
	isOfflineEvent := s.data.EventTypes.Has(events.EventType_DevicesOffline)
	switch {
	case isOnlineEvent && isOfflineEvent:
		return s.getDevicesOnlineOfflineEventHandler(emitEvent), nil
	case isOnlineEvent:
		return s.getDevicesOnlineEventHandler(emitEvent), nil
	case isOfflineEvent:
		return s.getDevicesOfflineEventHandler(emitEvent), nil
	}

	var eventHandler interface{}
	switch {
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesRegisteredUnregisteredHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		eventHandler = &devicesRegisteredHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesUnregisteredHandler{
			h: &devsHandler,
		}
	default:
		return nil, fmt.Errorf("createDevicesSubscription: unsupported subscription event types %+v", s.data.EventTypes)
	}
	return eventHandler, nil
}

func (s *SubscriptionData) createDevicesSubscription(ctx context.Context, emitEvent emitEventFunc, closeEventHandler *closeEventHandler) (Subscription, error) {
	if !s.data.Initialized {
		eventTypes, err := s.getEventsToEmitForDevicesSubscription(ctx)
		if err != nil {
			return nil, err
		}

		for _, e := range eventTypes {
			_, err := emitEvent(ctx, e, s.Data(), s.IncrementSequenceNumber, makeDevicesRepresentation([]string{}))
			if err != nil {
				return nil, err
			}
		}

		if err = s.SetInitialized(ctx); err != nil {
			return nil, err
		}
	}
	eventHandler, err := s.getEventHandlerForDevicesSubscription(emitEvent)
	if err != nil {
		return nil, err
	}
	return client.NewDevicesSubscription(ctx, closeEventHandler, eventHandler, s.gwClient)
}

func (s *SubscriptionData) createResourceSubscription(ctx context.Context, emitEvent emitEventFunc, closeEventHandler *closeEventHandler) (Subscription, error) {
	resHandler := resourceSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}
	var eventHandler interface{}
	switch {
	case s.data.EventTypes.Has(events.EventType_ResourceChanged):
		eventHandler = &resHandler
	default:
		return nil, fmt.Errorf("createResourceSubscription: unsupported subscription eventypes %+v", s.data.EventTypes)
	}
	return client.NewResourceSubscription(ctx, commands.NewResourceID(s.data.DeviceID, s.data.Href), closeEventHandler, eventHandler, s.gwClient)
}

func (s *SubscriptionData) createDeviceSubscription(ctx context.Context, emitEvent emitEventFunc, closeEventHandler *closeEventHandler) (Subscription, error) {
	devHandler := deviceSubscriptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}
	var eventHandler interface{}
	switch {
	case s.data.EventTypes.Has(events.EventType_ResourcesPublished) && s.data.EventTypes.Has(events.EventType_ResourcesUnpublished):
		eventHandler = &devHandler
	case s.data.EventTypes.Has(events.EventType_ResourcesPublished):
		eventHandler = &resourcePublishedHandler{
			h: &devHandler,
		}
	case s.data.EventTypes.Has(events.EventType_ResourcesUnpublished):
		eventHandler = &resourceUnpublishedHandler{
			h: &devHandler,
		}
	default:
		return nil, fmt.Errorf("createDeviceSubscription: unsupported subscription eventypes %+v", s.data.EventTypes)
	}
	return client.NewDeviceSubscription(ctx, s.data.DeviceID, closeEventHandler, eventHandler, s.gwClient)
}

func (s *SubscriptionData) Connect(ctx context.Context, emitEvent emitEventFunc, deleteSub func(ctx context.Context, subID string) (store.Subscription, error)) error {
	if s.Subscription() != nil {
		return fmt.Errorf("is already connected")
	}
	h := closeEventHandler{
		ctx:       ctx,
		deleteSub: deleteSub,
		data:      s,
		emitEvent: emitEvent,
	}

	var createSubscriptionFunc func(context.Context, emitEventFunc, *closeEventHandler) (Subscription, error)
	switch s.data.Type {
	case store.Type_Devices:
		createSubscriptionFunc = s.createDevicesSubscription
	case store.Type_Device:
		createSubscriptionFunc = s.createDeviceSubscription
	case store.Type_Resource:
		createSubscriptionFunc = s.createResourceSubscription
	default:
		return fmt.Errorf("unsupported subscription type: %v", store.Type_Device)
	}

	sub, err := createSubscriptionFunc(ctx, emitEvent, &h)
	if err != nil {
		if status.Convert(err).Code() == codes.Unauthenticated {
			subToCancel, errSub := deleteSub(ctx, s.data.ID)
			if errSub == nil {
				if err2 := cancelSubscription(ctx, emitEvent, subToCancel); err2 != nil {
					log.Errorf("cannot cancel subscription %v: %w", subToCancel.ID, err2)
				}
			}
		}
		return err
	}

	s.Store(sub)
	return nil
}

func (s *SubscriptionData) IncrementSequenceNumber(ctx context.Context) (uint64, error) {
	seqNum, err := s.incrementSubscriptionSequenceNumber(ctx, s.data.ID)
	if err != nil {
		return 0, err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data.SequenceNumber = seqNum
	return seqNum, nil
}

func (s *SubscriptionData) SetInitialized(ctx context.Context) error {
	return s.setInitialized(ctx, s.data.ID)
}

type closeEventHandler struct {
	ctx       context.Context
	emitEvent emitEventFunc
	deleteSub func(ctx context.Context, subID string) (store.Subscription, error)
	data      *SubscriptionData
}

func (h *closeEventHandler) OnClose() {
	log.Debugf("subscription %+v was closed", h.data.Data())
	h.data.Store(nil)
}

func (h *closeEventHandler) Error(err error) {
	data := h.data.Data()
	log.Errorf("subscription %+v ends with error: %w", data, err)
	if errors.Is(err, context.Canceled) {
		return
	}
	if !strings.Contains(err.Error(), "transport is closing") {
		sub, errSub := h.deleteSub(h.ctx, data.ID)
		if errSub == nil {
			if err2 := cancelSubscription(h.ctx, h.emitEvent, sub); err2 != nil {
				log.Errorf("cannot cancel subscription %v: %w", sub.ID, err2)
			}
		}
		return
	}
}

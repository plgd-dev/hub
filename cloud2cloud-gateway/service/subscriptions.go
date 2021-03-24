package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/store"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/log"
	kitSync "github.com/plgd-dev/kit/sync"
)

type Subscription interface {
	Cancel() (func(), error)
}

type SubscriptionData struct {
	incrementSubscriptionSequenceNumber func(ctx context.Context, subscriptionID string) (uint64, error)
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

func (s *SubscriptionData) createDevicesSubscription(ctx context.Context, emitEvent emitEventFunc, closeEventHandler *closeEventHandler) (Subscription, error) {
	devsHandler := devicesSubsciptionHandler{
		subData:   s,
		emitEvent: emitEvent,
	}
	var eventHandler interface{}
	switch {
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devsHandler
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		eventHandler = &devicesRegisteredOnlineOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesUnregisteredOnlineOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesRegisteredUnregisteredOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesRegisteredUnregisteredOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		eventHandler = &devicesRegisteredOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesUnregisteredOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOffline) && s.data.EventTypes.Has(events.EventType_DevicesOnline):
		eventHandler = &devicesOnlineOfflineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesRegistered):
		eventHandler = &devicesRegisteredOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOnline) && s.data.EventTypes.Has(events.EventType_DevicesUnregistered):
		eventHandler = &devicesUnregisteredOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOnline):
		eventHandler = &devicesOnlineHandler{
			h: &devsHandler,
		}
	case s.data.EventTypes.Has(events.EventType_DevicesOffline):
		eventHandler = &devicesOfflineHandler{
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
		return nil, fmt.Errorf("createDevicesSubsription: unsupported subscription eventypes %+v", s.data.EventTypes)
	}
	return client.NewDevicesSubscription(ctx, closeEventHandler, eventHandler, s.gwClient)
}

func (s *SubscriptionData) createResourceSubscription(ctx context.Context, emitEvent emitEventFunc, closeEventHandler *closeEventHandler) (Subscription, error) {
	resHandler := resourceSubsciptionHandler{
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
	devHandler := deviceSubsciptionHandler{
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
		return nil, fmt.Errorf("createDeviceSubsription: unsupported subscription eventypes %+v", s.data.EventTypes)
	}
	return client.NewDeviceSubscription(ctx, s.data.DeviceID, closeEventHandler, eventHandler, s.gwClient)
}

func (s *SubscriptionData) Connect(ctx context.Context, emitEvent emitEventFunc, deleteSub func(ctx context.Context, subID, userID string) (store.Subscription, error)) error {
	if s.Subscription() != nil {
		return fmt.Errorf("is already connected")
	}
	closeEventHandler := closeEventHandler{
		ctx:       ctx,
		deleteSub: deleteSub,
		data:      s,
		emitEvent: emitEvent,
	}

	ctx = kitNetGrpc.CtxWithOwner(ctx, s.Data().UserID)
	var err error
	var sub Subscription
	switch s.data.Type {
	case store.Type_Devices:
		sub, err = s.createDevicesSubscription(ctx, emitEvent, &closeEventHandler)
		if err != nil {
			return err
		}
	case store.Type_Device:
		sub, err = s.createDeviceSubscription(ctx, emitEvent, &closeEventHandler)
		if err != nil {
			return err
		}
	case store.Type_Resource:
		sub, err = s.createResourceSubscription(ctx, emitEvent, &closeEventHandler)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported subscription type: %v", store.Type_Device)
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

type closeEventHandler struct {
	ctx       context.Context
	emitEvent emitEventFunc
	deleteSub func(ctx context.Context, subID, userID string) (store.Subscription, error)
	data      *SubscriptionData
}

func (h *closeEventHandler) OnClose() {
	log.Errorf("subscription %+v was closed", h.data.Data())
	h.data.Store(nil)
}

func (h *closeEventHandler) Error(err error) {
	data := h.data.Data()
	log.Errorf("subscription %+v ends with error: %v", data, err)
	if errors.Is(err, context.Canceled) {
		return
	}
	if !strings.Contains(err.Error(), "transport is closing") {
		sub, errSub := h.deleteSub(h.ctx, data.ID, data.UserID)
		if errSub == nil {
			cancelSubscription(h.ctx, h.emitEvent, sub)
		}
		return
	}
}

type SubscriptionManager struct {
	ctx               context.Context
	subscriptions     *kitSync.Map
	store             store.Store
	gwClient          pb.GrpcGatewayClient
	reconnectInterval time.Duration
	emitEvent         emitEventFunc
}

func NewSubscriptionManager(ctx context.Context, store store.Store, gwClient pb.GrpcGatewayClient, reconnectInterval time.Duration, emitEvent emitEventFunc) *SubscriptionManager {
	return &SubscriptionManager{
		store:             store,
		reconnectInterval: reconnectInterval,
		subscriptions:     kitSync.NewMap(),
		gwClient:          gwClient,
		ctx:               ctx,
		emitEvent:         emitEvent,
	}
}

type subscriptionLoader struct {
	s *SubscriptionManager
}

func (l *subscriptionLoader) Handle(ctx context.Context, iter store.SubscriptionIter) error {
	for {
		var s store.Subscription
		if !iter.Next(ctx, &s) {
			break
		}
		l.s.storeToSubs(s)
	}
	return iter.Err()
}

func (s *SubscriptionManager) LoadSubscriptions() error {
	h := subscriptionLoader{
		s: s,
	}
	err := s.store.LoadSubscriptions(s.ctx, store.SubscriptionQuery{}, &h)
	if err != nil {
		return err
	}
	return nil
}

func (s *SubscriptionManager) storeToSubs(sub store.Subscription) {
	subData := &SubscriptionData{
		data:                                sub,
		incrementSubscriptionSequenceNumber: s.store.IncrementSubscriptionSequenceNumber,
		gwClient:                            s.gwClient,
	}
	s.subscriptions.LoadOrStore(sub.ID, subData)
}

func (s *SubscriptionManager) Connect(ID string) error {
	subRaw, ok := s.subscriptions.Load(ID)
	if !ok {
		return fmt.Errorf("not found")
	}
	sub := subRaw.(*SubscriptionData)
	if sub.sub != nil {
		if !ok {
			return fmt.Errorf("already connected")
		}
	}
	return sub.Connect(s.ctx, s.emitEvent, s.PullOut)
}

func (s *SubscriptionManager) Store(ctx context.Context, sub store.Subscription) error {
	err := s.store.SaveSubscription(ctx, sub)
	if err != nil {
		return err
	}
	s.storeToSubs(sub)
	return nil
}

func (s *SubscriptionManager) Load(ID, userID string) (store.Subscription, bool) {
	subDataRaw, ok := s.subscriptions.Load(ID)
	if !ok {
		return store.Subscription{}, false
	}
	subData := subDataRaw.(*SubscriptionData)
	data := subData.Data()
	if data.UserID != userID {
		return store.Subscription{}, false
	}
	return data, true
}

func cancelSubscription(ctx context.Context, emitEvent emitEventFunc, sub store.Subscription) error {
	_, err := emitEvent(ctx, events.EventType_SubscriptionCanceled, sub, func(ctx context.Context) (uint64, error) {
		return sub.SequenceNumber, nil
	}, nil)
	return err
}

func (s *SubscriptionManager) PullOut(ctx context.Context, ID, userID string) (store.Subscription, error) {
	var found bool
	subDataRaw, ok := s.subscriptions.ReplaceWithFunc(ID, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
		if !oldLoaded {
			return nil, true
		}
		data := oldValue.(*SubscriptionData)
		if data.Data().UserID != userID {
			return oldValue, false
		}
		found = true
		return nil, true
	})
	if !ok || !found {
		return store.Subscription{}, fmt.Errorf("not found")
	}
	sub, err := s.store.PopSubscription(ctx, ID)
	if err != nil {
		return store.Subscription{}, err
	}
	subData := subDataRaw.(*SubscriptionData)
	subscription := subData.Subscription()
	if subscription == nil {
		wait, err := subscription.Cancel()
		if err == nil {
			wait()
		}
	}
	return sub, nil
}

func (s *SubscriptionManager) DumpNotConnectedSubscriptionDatas() map[string]*SubscriptionData {
	out := make(map[string]*SubscriptionData)
	s.subscriptions.Range(func(key, resourceI interface{}) bool {
		subData := resourceI.(*SubscriptionData)
		if subData.Subscription() == nil {
			out[key.(string)] = resourceI.(*SubscriptionData)
		}
		return true
	})
	return out
}

func (s *SubscriptionManager) Run() {
	for {
		var wg sync.WaitGroup
		for _, task := range s.DumpNotConnectedSubscriptionDatas() {
			wg.Add(1)
			go func(subData *SubscriptionData) {
				defer wg.Done()
				err := subData.Connect(s.ctx, s.emitEvent, s.PullOut)
				if err != nil {
					log.Errorf("cannot connect %+v: %v", subData.Data(), err)
				}
			}(task)
		}
		wg.Wait()
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(s.reconnectInterval):
		}
	}
}

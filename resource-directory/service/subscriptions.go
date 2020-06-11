package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc/codes"

	clientAS "github.com/go-ocf/cloud/authorization/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema/cloud"
	"github.com/gofrs/uuid"

	"github.com/go-ocf/cqrs/eventstore"
)

type Subscriber interface {
	UserID() string
	ID() string
	Init(ctx context.Context, currentDevices map[string]bool) error
	Close(reason error) error
}

type subscriptions struct {
	userDevicesManager *clientAS.UserDevicesManager

	rwlock                sync.RWMutex
	allSubscriptions      map[string]Subscriber                                             // map[subscriptionID]
	resourceSubscriptions map[string]map[string]map[string]map[string]*resourceSubscription // map[userId]map[req.DeviceId]map[href]map[subscriptionID]
	deviceSubscriptions   map[string]map[string]map[string]*deviceSubscription              // map[userId]map[req.DeviceId]map[subscriptionID]
	devicesSubscriptions  map[string]map[string]*devicesSubscription                        // map[userId]map[subscriptionID]

	initSubscriptionsLock sync.Mutex
	initSubscriptions     map[string]map[string]Subscriber // map[userId]map[subscriptionID]
}

type SendEventFunc func(senderCtx context.Context, e pb.Event) error

func NewSubscriptions() *subscriptions {
	return &subscriptions{
		allSubscriptions:      make(map[string]Subscriber),
		resourceSubscriptions: make(map[string]map[string]map[string]map[string]*resourceSubscription),
		deviceSubscriptions:   make(map[string]map[string]map[string]*deviceSubscription),
		devicesSubscriptions:  make(map[string]map[string]*devicesSubscription),
		initSubscriptions:     make(map[string]map[string]Subscriber),
	}
}

func (s *subscriptions) popInitSubscriptions(userID string) map[string]Subscriber {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()
	v := s.initSubscriptions[userID]
	delete(s.initSubscriptions, userID)
	return v
}

func (s *subscriptions) insertToInitSubscriptions(sub Subscriber) {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()

	v, ok := s.initSubscriptions[sub.UserID()]
	if !ok {
		v = make(map[string]Subscriber)
		s.initSubscriptions[sub.UserID()] = v
	}
	v[sub.ID()] = sub
}

func (s *subscriptions) removeFromInitSubscriptions(id, userID string) {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()

	delete(s.initSubscriptions[userID], id)
	if len(s.initSubscriptions[userID]) == 0 {
		delete(s.initSubscriptions, userID)
	}
}

func (s *subscriptions) getRemovedSubscriptions(userID string, removedDevices map[string]bool) []Subscriber {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	remove := make([]Subscriber, 0, 32)
	for deviceID := range removedDevices {
		for _, sub := range s.deviceSubscriptions[userID][deviceID] {
			remove = append(remove, sub)
		}
		for _, resSubs := range s.resourceSubscriptions[userID][deviceID] {
			for _, sub := range resSubs {
				remove = append(remove, sub)
			}
		}
	}

	return remove
}

func (s *subscriptions) getSubscriptionsToUpdate(userID string, init map[string]Subscriber) []*devicesSubscription {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	updated := make([]*devicesSubscription, 0, 32)
	for _, sub := range s.devicesSubscriptions[userID] {
		if _, ok := init[sub.ID()]; !ok {
			updated = append(updated, sub)
		}
	}

	return updated
}

func (s *subscriptions) UserDevicesChanged(ctx context.Context, userID string, addedDevices, removedDevices, currentDevices map[string]bool) {
	log.Debugf("subscriptions.UserDevicesChanged %v: added: %+v removed: %+v current: %+v\n", userID, addedDevices, removedDevices, currentDevices)

	init := s.popInitSubscriptions(userID)
	for _, sub := range init {
		err := sub.Init(ctx, currentDevices)
		if err != nil {
			log.Errorf("cannot init sub ID %v: %v", sub.ID(), err)
			s.Close(sub.ID(), err)
		}
	}
	remove := s.getRemovedSubscriptions(userID, removedDevices)
	for _, sub := range remove {
		log.Infof("remove sub ID %v", sub.ID())
		sub.Close(fmt.Errorf("device was removed from user"))
	}

	if len(addedDevices) > 0 || len(removedDevices) > 0 {
		update := s.getSubscriptionsToUpdate(userID, init)
		for _, sub := range update {
			err := sub.Update(ctx, addedDevices, removedDevices)
			if err != nil {
				log.Errorf("cannot update sub ID %v: %v", sub.ID(), err)
				s.Close(sub.ID(), err)
			}
		}
	}
}

func (s *subscriptions) Get(id string) Subscriber {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	if sub, ok := s.allSubscriptions[id]; ok {
		return sub
	}
	return nil
}

func (s *subscriptions) Pop(id string) Subscriber {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	if sub, ok := s.allSubscriptions[id]; ok {
		userID := sub.UserID()
		switch v := sub.(type) {
		case *deviceSubscription:
			delete(s.deviceSubscriptions[userID][v.DeviceID()], id)
			if len(s.deviceSubscriptions[userID][v.DeviceID()]) == 0 {
				delete(s.deviceSubscriptions, v.DeviceID())
				if len(s.deviceSubscriptions[userID]) == 0 {
					delete(s.deviceSubscriptions, userID)
				}
			}
		case *resourceSubscription:
			deviceID := v.DeviceID()
			href := v.Href()
			delete(s.resourceSubscriptions[userID][deviceID][href], id)
			if len(s.resourceSubscriptions[userID][deviceID][href]) == 0 {
				delete(s.resourceSubscriptions[userID][deviceID], href)
				if len(s.resourceSubscriptions[userID][deviceID]) == 0 {
					delete(s.resourceSubscriptions[userID], deviceID)
					if len(s.resourceSubscriptions[userID]) == 0 {
						delete(s.resourceSubscriptions, userID)
					}
				}
			}
		case *devicesSubscription:
			delete(s.devicesSubscriptions[userID], id)
			if len(s.devicesSubscriptions[userID]) == 0 {
				delete(s.devicesSubscriptions, userID)
			}
		}
		delete(s.allSubscriptions, id)
		return sub
	}
	return nil
}

func (s *subscriptions) closeWithReleaseUserDevicesMfg(id string, reason error, releaseFromUserDevManager bool) error {
	sub := s.Pop(id)
	if sub == nil {
		return fmt.Errorf("cannot close subscription %v: not found", id)
	}
	s.removeFromInitSubscriptions(id, sub.UserID())
	if releaseFromUserDevManager {
		s.userDevicesManager.Release(sub.UserID())
	}

	err := sub.Close(reason)
	if err != nil {
		return fmt.Errorf("cannot close subscription %v: %w", id, err)
	}
	return nil
}

func (s *subscriptions) Close(id string, reason error) error {
	return s.closeWithReleaseUserDevicesMfg(id, reason, true)
}

func (s *subscriptions) InsertDevicesSubscription(ctx context.Context, sub *devicesSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	userID := sub.UserID()
	userSubs, ok := s.devicesSubscriptions[userID]
	if !ok {
		userSubs = make(map[string]*devicesSubscription)
		s.devicesSubscriptions[userID] = userSubs
	}
	userSubs[sub.ID()] = sub

	initSubs, ok := s.initSubscriptions[userID]
	if !ok {
		initSubs = make(map[string]Subscriber)
		s.initSubscriptions[userID] = initSubs
	}
	initSubs[sub.ID()] = sub
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *subscriptions) InsertDeviceSubscription(ctx context.Context, sub *deviceSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	userID := sub.UserID()
	deviceID := sub.DeviceID()
	userSubs, ok := s.deviceSubscriptions[userID]
	if !ok {
		userSubs = make(map[string]map[string]*deviceSubscription)
		s.deviceSubscriptions[userID] = userSubs
	}
	devSubs, ok := userSubs[deviceID]
	if !ok {
		devSubs = make(map[string]*deviceSubscription)
		userSubs[deviceID] = devSubs
	}
	devSubs[sub.ID()] = sub

	initSubs, ok := s.initSubscriptions[userID]
	if !ok {
		initSubs = make(map[string]Subscriber)
		s.initSubscriptions[userID] = initSubs
	}
	initSubs[sub.ID()] = sub
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *subscriptions) InsertResourceSubscription(ctx context.Context, sub *resourceSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	userID := sub.UserID()
	deviceID := sub.DeviceID()
	href := sub.Href()
	userSubs, ok := s.resourceSubscriptions[userID]
	if !ok {
		userSubs = make(map[string]map[string]map[string]*resourceSubscription)
		s.resourceSubscriptions[userID] = userSubs
	}
	devSubs, ok := userSubs[deviceID]
	if !ok {
		devSubs = make(map[string]map[string]*resourceSubscription)
		userSubs[deviceID] = devSubs
	}
	resSubs, ok := devSubs[href]
	if !ok {
		resSubs = make(map[string]*resourceSubscription)
		devSubs[href] = resSubs
	}
	resSubs[sub.ID()] = sub

	initSubs, ok := s.initSubscriptions[userID]
	if !ok {
		initSubs = make(map[string]Subscriber)
		s.initSubscriptions[userID] = initSubs
	}
	initSubs[sub.ID()] = sub
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func makeLinkRepresentation(eventType pb.SubscribeForEvents_DeviceEventFilter_Event, m eventstore.Model) (pb.ResourceLink, uint64, bool) {
	c := m.(*resourceCtx).Clone()
	switch eventType {
	case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED:
		if c.isPublished {
			return pb.RAResourceToProto(c.resource), c.onResourcePublishedVersion, true
		}
	case pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED:
		if !c.isPublished {
			return pb.RAResourceToProto(c.resource), c.onResourceUnpublishedVersion, true
		}
	}
	return pb.ResourceLink{}, 0, false
}

func makeContent(content *pbRA.Content) pb.Content {
	return pb.Content{
		ContentType: content.ContentType,
		Data:        content.Data,
	}
}

func (s *subscriptions) OnResourcePublished(ctx context.Context, link pb.ResourceLink, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, link.DeviceId) {
			continue
		}
		for _, sub := range userSubs[link.DeviceId] {
			if err := sub.NotifyOfPublishedResource(ctx, link, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource published event: %v", errors)
	}
	return nil
}

func (s *subscriptions) OnResourceUnpublished(ctx context.Context, link pb.ResourceLink, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, link.DeviceId) {
			continue
		}
		for _, sub := range userSubs[link.DeviceId] {
			if err := sub.NotifyOfUnpublishedResource(ctx, link, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource unpublished event: %v", errors)
	}
	return nil
}

func (s *subscriptions) OnResourceUpdatePending(ctx context.Context, updatePending pb.Event_ResourceUpdatePending, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, updatePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[updatePending.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfUpdatePendingResource(ctx, updatePending, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource update pending event: %v", errors)
	}
	return nil
}

func (s *subscriptions) OnResourceUpdated(ctx context.Context, updated pb.Event_ResourceUpdated, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, updated.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[updated.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfUpdatedResource(ctx, updated, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource updated event: %v", errors)
	}
	return nil
}

func (s *subscriptions) OnDeviceOnline(ctx context.Context, deviceID string, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.devicesSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, deviceID) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfOnlineDevice(ctx, deviceID, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device online event: %v", errors)
	}

	return nil
}

func (s *subscriptions) OnDeviceOffline(ctx context.Context, deviceID string, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.devicesSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, deviceID) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfOfflineDevice(ctx, deviceID, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device offline event: %v", errors)
	}
	return nil
}

func (s *subscriptions) OnResourceContentChanged(ctx context.Context, deviceID, href string, content pb.Content, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for userID, userSubs := range s.resourceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(userID, deviceID) {
			continue
		}
		res, ok := userSubs[deviceID]
		if !ok {
			return nil
		}
		subs := res[href]
		for _, sub := range subs {
			if err := sub.NotifyOfContentChangedResource(ctx, content, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource content changed: %v", errors)
	}
	return nil
}

func (s *subscriptions) CancelResourceSubscriptions(ctx context.Context, deviceID, href string, err error) {
	subsIDs := make([]string, 0, 4)
	func() {
		s.rwlock.RLock()
		defer s.rwlock.RUnlock()
		for _, userSubs := range s.resourceSubscriptions {
			subs := userSubs[deviceID][href]
			for subID := range subs {
				subsIDs = append(subsIDs, subID)
			}
		}
	}()

	for _, id := range subsIDs {
		s.Close(id, err)
	}
}

func isDeviceOnline(content *pbRA.Content) (bool, error) {
	if content == nil {
		return false, nil
	}
	var decoder func(data []byte, v interface{}) error
	switch content.ContentType {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decoder = cbor.Decode
	case message.AppJSON.String():
		decoder = json.Decode
	}
	if decoder == nil {
		return false, fmt.Errorf("decoder not found")
	}
	var cloudStatus cloud.Status
	err := decoder(content.Data, &cloudStatus)
	if err != nil {
		return false, err
	}
	return cloudStatus.Online, nil
}

func (s *subscriptions) SubscribeForDevicesEvent(ctx context.Context, userID string, resourceProjection *Projection, subscriptionID string, send SendEventFunc, req *pb.SubscribeForEvents_DevicesEventFilter) error {
	sub := NewDevicesSubscription(subscriptionID, userID, send, resourceProjection, req)
	err := s.InsertDevicesSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, userID)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *subscriptions) SubscribeForDeviceEvent(ctx context.Context, userID string, resourceProjection *Projection, subscriptionID string, send SendEventFunc, req *pb.SubscribeForEvents_DeviceEventFilter) error {
	sub := NewDeviceSubscription(subscriptionID, userID, send, resourceProjection, req)
	err := s.InsertDeviceSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, userID)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *subscriptions) SubscribeForResourceEvent(ctx context.Context, userID string, resourceProjection *Projection, subscriptionID string, send SendEventFunc, req *pb.SubscribeForEvents_ResourceEventFilter) error {
	sub := NewResourceSubscription(subscriptionID, userID, send, resourceProjection, req)
	err := s.InsertResourceSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, userID)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *subscriptions) cancelSubscription(localSubscriptions *sync.Map, subscriptionID string) error {
	_, ok := localSubscriptions.Load(subscriptionID)
	if !ok {
		return fmt.Errorf("cannot cancel subscription %v: not found", subscriptionID)
	}
	localSubscriptions.Delete(subscriptionID)
	return s.Close(subscriptionID, nil)
}

func (s *subscriptions) SubscribeForEvents(resourceProjection *Projection, srv pb.GrpcGateway_SubscribeForEventsServer) error {
	userID, err := kitNetGrpc.UserIDFromMD(srv.Context())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	var wg sync.WaitGroup
	var localSubscriptions sync.Map

	defer wg.Wait()
	wg.Add(1)
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	defer func() {
		subs := make([]string, 0, 32)
		localSubscriptions.Range(func(k interface{}, _ interface{}) bool {
			subs = append(subs, k.(string))
			return true
		})

		for _, sub := range subs {
			err := s.Close(sub, nil)
			if err != nil {
				log.Errorf("cannot close subscription for events: %v", err)
			}
		}
	}()

	sendChan := make(chan pb.Event, 16)
	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-sendChan:
				err := srv.Send(&e)
				if err != nil {
					log.Errorf("cannot send event %+v: %v", e, err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	send := func(senderCtx context.Context, e pb.Event) error {
		log.Debugf("subscriptions.SubscribeForEvents.send: %v %+v", e.GetSubscriptionId(), e.GetType())

		select {
		case sendChan <- e:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("cannot send event: stream context returns error: %v", ctx.Err())
		case <-senderCtx.Done():
			return fmt.Errorf("cannot send event: sender context returns error: %v", ctx.Err())
		}
	}

	for {
		subReq, err := srv.Recv()
		if err == io.EOF {
			log.Debugf("subscriptions.SubscribeForEvents: cannot receive events for userID %v: %v", userID, err)
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err)
		}

		subRes := pb.Event{
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					Token: subReq.Token,
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code: pb.Event_OperationProcessed_ErrorStatus_OK,
					},
				},
			},
		}

		if r := subReq.GetCancelSubscription(); r != nil {
			err := s.cancelSubscription(&localSubscriptions, r.GetSubscriptionId())
			if err != nil {
				subRes.GetOperationProcessed().ErrorStatus.Code = pb.Event_OperationProcessed_ErrorStatus_ERROR
				subRes.GetOperationProcessed().ErrorStatus.Message = err.Error()
			}
			subRes.SubscriptionId = r.GetSubscriptionId()
			send(ctx, subRes)
			continue
		}

		subID, err := uuid.NewV4()
		if err != nil {
			subRes.GetOperationProcessed().ErrorStatus.Code = pb.Event_OperationProcessed_ErrorStatus_ERROR
			subRes.GetOperationProcessed().ErrorStatus.Message = fmt.Sprintf("cannot generate subscription ID: %v", err)
			send(ctx, subRes)
			continue
		}

		subRes.SubscriptionId = subID.String()
		localSubscriptions.Store(subRes.SubscriptionId, true)
		send(ctx, subRes)

		switch r := subReq.GetFilterBy().(type) {
		case *pb.SubscribeForEvents_DevicesEvent:
			err = s.SubscribeForDevicesEvent(ctx, userID, resourceProjection, subRes.SubscriptionId, send, r.DevicesEvent)
		case *pb.SubscribeForEvents_DeviceEvent:
			err = s.SubscribeForDeviceEvent(ctx, userID, resourceProjection, subRes.SubscriptionId, send, r.DeviceEvent)
		case *pb.SubscribeForEvents_ResourceEvent:
			err = s.SubscribeForResourceEvent(ctx, userID, resourceProjection, subRes.SubscriptionId, send, r.ResourceEvent)
		case *pb.SubscribeForEvents_CancelSubscription_:
			//handled by cancelation
			err = nil
		default:
			err = fmt.Errorf("not supported")
			send(ctx, pb.Event{
				SubscriptionId: subRes.SubscriptionId,
				Type: &pb.Event_SubscriptionCanceled_{
					SubscriptionCanceled: &pb.Event_SubscriptionCanceled{
						Reason: err.Error(),
					},
				}})
		}

		if err != nil {
			localSubscriptions.Delete(subRes.SubscriptionId)
			log.Errorf("errors occurs during %T: %v", subReq.GetFilterBy(), err)
		}
	}
	return nil
}

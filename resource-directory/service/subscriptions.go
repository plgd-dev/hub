package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc/codes"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	"github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
)

type Subscriber interface {
	UserID() string
	ID() string
	Init(ctx context.Context, currentDevices map[string]bool) error
	Close(reason error) error
}

type Subscriptions struct {
	userDevicesManager *clientAS.UserDevicesManager

	rwlock                sync.RWMutex
	allSubscriptions      map[string]Subscriber                                             // map[subscriptionID]
	resourceSubscriptions map[string]map[string]map[string]map[string]*resourceSubscription // map[userId]map[req.DeviceId]map[href]map[subscriptionID]
	deviceSubscriptions   map[string]map[string]map[string]*deviceSubscription              // map[userId]map[req.DeviceId]map[subscriptionID]
	devicesSubscriptions  map[string]map[string]*devicesSubscription                        // map[userId]map[subscriptionID]

	initSubscriptionsLock sync.Mutex
	initSubscriptions     map[string]map[string]Subscriber // map[userId]map[subscriptionID]
}

type SendEventFunc func(e *pb.Event) error

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		allSubscriptions:      make(map[string]Subscriber),
		resourceSubscriptions: make(map[string]map[string]map[string]map[string]*resourceSubscription),
		deviceSubscriptions:   make(map[string]map[string]map[string]*deviceSubscription),
		devicesSubscriptions:  make(map[string]map[string]*devicesSubscription),
		initSubscriptions:     make(map[string]map[string]Subscriber),
	}
}

func (s *Subscriptions) popInitSubscriptions(owner string) map[string]Subscriber {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()
	v := s.initSubscriptions[owner]
	delete(s.initSubscriptions, owner)
	return v
}

func (s *Subscriptions) insertToInitSubscriptions(sub Subscriber) {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()

	v, ok := s.initSubscriptions[sub.UserID()]
	if !ok {
		v = make(map[string]Subscriber)
		s.initSubscriptions[sub.UserID()] = v
	}
	v[sub.ID()] = sub
}

func (s *Subscriptions) removeFromInitSubscriptions(id, owner string) {
	s.initSubscriptionsLock.Lock()
	defer s.initSubscriptionsLock.Unlock()

	delete(s.initSubscriptions[owner], id)
	if len(s.initSubscriptions[owner]) == 0 {
		delete(s.initSubscriptions, owner)
	}
}

func (s *Subscriptions) getRemovedSubscriptions(owner string, removedDevices map[string]bool) []Subscriber {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	remove := make([]Subscriber, 0, 32)
	for deviceID := range removedDevices {
		for _, sub := range s.deviceSubscriptions[owner][deviceID] {
			remove = append(remove, sub)
		}
		for _, resSubs := range s.resourceSubscriptions[owner][deviceID] {
			for _, sub := range resSubs {
				remove = append(remove, sub)
			}
		}
	}

	return remove
}

func (s *Subscriptions) getSubscriptionsToUpdate(owner string, init map[string]Subscriber) []*devicesSubscription {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	updated := make([]*devicesSubscription, 0, 32)
	for _, sub := range s.devicesSubscriptions[owner] {
		if _, ok := init[sub.ID()]; !ok {
			updated = append(updated, sub)
		}
	}

	return updated
}

func (s *Subscriptions) UserDevicesChanged(ctx context.Context, owner string, addedDevices, removedDevices, currentDevices map[string]bool) {
	log.Debugf("subscriptions.UserDevicesChanged %v: added: %+v removed: %+v current: %+v\n", owner, addedDevices, removedDevices, currentDevices)

	init := s.popInitSubscriptions(owner)
	for _, sub := range init {
		err := sub.Init(ctx, currentDevices)
		if err != nil {
			log.Errorf("cannot init sub ID %v: %v", sub.ID(), err)
			s.Close(sub.ID(), err)
		}
	}
	remove := s.getRemovedSubscriptions(owner, removedDevices)
	for _, sub := range remove {
		log.Infof("remove sub ID %v", sub.ID())
		sub.Close(fmt.Errorf("device was removed from user"))
	}

	if len(addedDevices) > 0 || len(removedDevices) > 0 {
		update := s.getSubscriptionsToUpdate(owner, init)
		for _, sub := range update {
			err := sub.Update(ctx, addedDevices, removedDevices)
			if err != nil {
				log.Errorf("cannot update sub ID %v: %v", sub.ID(), err)
				s.Close(sub.ID(), err)
			}
		}
	}
}

func (s *Subscriptions) Get(id string) Subscriber {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	if sub, ok := s.allSubscriptions[id]; ok {
		return sub
	}
	return nil
}

func (s *Subscriptions) Pop(id string) Subscriber {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	if sub, ok := s.allSubscriptions[id]; ok {
		owner := sub.UserID()
		switch v := sub.(type) {
		case *deviceSubscription:
			delete(s.deviceSubscriptions[owner][v.DeviceID()], id)
			if len(s.deviceSubscriptions[owner][v.DeviceID()]) == 0 {
				delete(s.deviceSubscriptions, v.DeviceID())
				if len(s.deviceSubscriptions[owner]) == 0 {
					delete(s.deviceSubscriptions, owner)
				}
			}
		case *resourceSubscription:
			deviceID := v.ResourceID().GetDeviceId()
			href := v.ResourceID().GetHref()
			delete(s.resourceSubscriptions[owner][deviceID][href], id)
			if len(s.resourceSubscriptions[owner][deviceID][href]) == 0 {
				delete(s.resourceSubscriptions[owner][deviceID], href)
				if len(s.resourceSubscriptions[owner][deviceID]) == 0 {
					delete(s.resourceSubscriptions[owner], deviceID)
					if len(s.resourceSubscriptions[owner]) == 0 {
						delete(s.resourceSubscriptions, owner)
					}
				}
			}
		case *devicesSubscription:
			delete(s.devicesSubscriptions[owner], id)
			if len(s.devicesSubscriptions[owner]) == 0 {
				delete(s.devicesSubscriptions, owner)
			}
		}
		delete(s.allSubscriptions, id)
		return sub
	}
	return nil
}

func (s *Subscriptions) closeWithReleaseUserDevicesMfg(id string, reason error, releaseFromUserDevManager bool) error {
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

func (s *Subscriptions) Close(id string, reason error) error {
	return s.closeWithReleaseUserDevicesMfg(id, reason, true)
}

func (s *Subscriptions) InsertDevicesSubscription(ctx context.Context, sub *devicesSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	owner := sub.UserID()
	userSubs, ok := s.devicesSubscriptions[owner]
	if !ok {
		userSubs = make(map[string]*devicesSubscription)
		s.devicesSubscriptions[owner] = userSubs
	}
	userSubs[sub.ID()] = sub

	s.insertToInitSubscriptions(sub)
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *Subscriptions) InsertDeviceSubscription(ctx context.Context, sub *deviceSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	owner := sub.UserID()
	deviceID := sub.DeviceID()
	userSubs, ok := s.deviceSubscriptions[owner]
	if !ok {
		userSubs = make(map[string]map[string]*deviceSubscription)
		s.deviceSubscriptions[owner] = userSubs
	}
	devSubs, ok := userSubs[deviceID]
	if !ok {
		devSubs = make(map[string]*deviceSubscription)
		userSubs[deviceID] = devSubs
	}
	devSubs[sub.ID()] = sub

	s.insertToInitSubscriptions(sub)
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *Subscriptions) InsertResourceSubscription(ctx context.Context, sub *resourceSubscription) error {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()
	_, ok := s.allSubscriptions[sub.ID()]
	if ok {
		return fmt.Errorf("subscription already exist")
	}
	if sub == nil {
		return nil
	}
	owner := sub.UserID()
	deviceID := sub.ResourceID().GetDeviceId()
	href := sub.ResourceID().GetHref()
	userSubs, ok := s.resourceSubscriptions[owner]
	if !ok {
		userSubs = make(map[string]map[string]map[string]*resourceSubscription)
		s.resourceSubscriptions[owner] = userSubs
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
	s.insertToInitSubscriptions(sub)
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *Subscriptions) OnResourceLinksPublished(ctx context.Context, deviceID string, links ResourceLinks) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deviceID) {
			continue
		}
		for _, sub := range userSubs[deviceID] {
			if err := sub.NotifyOfPublishedResourceLinks(ctx, links); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource published event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceLinksUnpublished(ctx context.Context, deviceID string, links ResourceLinks) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deviceID) {
			continue
		}
		for _, sub := range userSubs[deviceID] {
			if err := sub.NotifyOfUnpublishedResourceLinks(ctx, links); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource unpublished event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceUpdatePending(ctx context.Context, updatePending *events.ResourceUpdatePending, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, updatePending.GetResourceId().GetDeviceId()) {
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

func (s *Subscriptions) OnResourceUpdated(ctx context.Context, updated *events.ResourceUpdated, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, updated.GetResourceId().GetDeviceId()) {
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

func (s *Subscriptions) OnResourceRetrievePending(ctx context.Context, retrievePending *events.ResourceRetrievePending, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, retrievePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[retrievePending.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfRetrievePendingResource(ctx, retrievePending, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource retrieve pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceDeletePending(ctx context.Context, deletePending *events.ResourceDeletePending, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deletePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[deletePending.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfDeletePendingResource(ctx, deletePending, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource delete pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceCreatePending(ctx context.Context, createPending *events.ResourceCreatePending, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, createPending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[createPending.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfCreatePendingResource(ctx, createPending, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource create pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceRetrieved(ctx context.Context, retrieved *events.ResourceRetrieved, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, retrieved.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[retrieved.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfRetrievedResource(ctx, retrieved, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource retrieved event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceDeleted(ctx context.Context, deleted *events.ResourceDeleted, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deleted.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[deleted.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfDeletedResource(ctx, deleted, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource deleted event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceCreated(ctx context.Context, created *events.ResourceCreated, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.deviceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, created.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs[created.GetResourceId().GetDeviceId()] {
			if err := sub.NotifyOfCreatedResource(ctx, created, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource created event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnDeviceOnline(ctx context.Context, dev DeviceIDVersion) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.devicesSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, dev.deviceID) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfOnlineDevice(ctx, []DeviceIDVersion{dev}); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device online event: %v", errors)
	}

	return nil
}

func (s *Subscriptions) OnDeviceOffline(ctx context.Context, dev DeviceIDVersion) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.devicesSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, dev.deviceID) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfOfflineDevice(ctx, []DeviceIDVersion{dev}); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device offline event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceContentChanged(ctx context.Context, resourceChanged *pb.Event_ResourceChanged, version uint64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	deviceID := resourceChanged.GetResourceId().GetDeviceId()
	href := resourceChanged.GetResourceId().GetHref()
	for owner, userSubs := range s.resourceSubscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deviceID) {
			continue
		}
		res, ok := userSubs[deviceID]
		if !ok {
			return nil
		}
		subs := res[href]
		for _, sub := range subs {
			if err := sub.NotifyOfContentChangedResource(ctx, resourceChanged, version); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource content changed: %v", errors)
	}
	return nil
}

func (s *Subscriptions) CancelResourceSubscriptions(ctx context.Context, deviceID, href string, err error) {
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

func isDeviceOnline(content *commands.Content) (bool, error) {
	if content == nil {
		return false, nil
	}
	var decoder func(data []byte, v interface{}) error
	switch content.GetContentType() {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		decoder = cbor.Decode
	case message.AppJSON.String():
		decoder = json.Decode
	}
	if decoder == nil {
		return false, fmt.Errorf("decoder not found")
	}
	var cloudStatus status.Status
	err := decoder(content.GetData(), &cloudStatus)
	if err != nil {
		return false, err
	}
	return cloudStatus.IsOnline(), nil
}

func (s *Subscriptions) SubscribeForDevicesEvent(ctx context.Context, owner string, resourceProjection *Projection, subscriptionID, token string, send SendEventFunc, req *pb.SubscribeForEvents_DevicesEventFilter) error {
	sub := NewDevicesSubscription(subscriptionID, owner, token, send, resourceProjection, req)
	err := s.InsertDevicesSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, owner)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *Subscriptions) SubscribeForDeviceEvent(ctx context.Context, owner string, resourceProjection *Projection, subscriptionID, token string, send SendEventFunc, req *pb.SubscribeForEvents_DeviceEventFilter) error {
	sub := NewDeviceSubscription(subscriptionID, owner, token, send, resourceProjection, req)
	err := s.InsertDeviceSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, owner)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *Subscriptions) SubscribeForResourceEvent(ctx context.Context, owner string, resourceProjection *Projection, subscriptionID, token string, send SendEventFunc, req *pb.SubscribeForEvents_ResourceEventFilter) error {
	sub := NewResourceSubscription(subscriptionID, owner, token, send, resourceProjection, req)
	err := s.InsertResourceSubscription(ctx, sub)
	if err != nil {
		sub.Close(err)
		return err
	}
	err = s.userDevicesManager.Acquire(ctx, owner)
	if err != nil {
		s.closeWithReleaseUserDevicesMfg(subscriptionID, err, false)
		return err
	}
	return nil
}

func (s *Subscriptions) cancelSubscription(localSubscriptions *sync.Map, subscriptionID string) error {
	_, ok := localSubscriptions.Load(subscriptionID)
	if !ok {
		return fmt.Errorf("cannot cancel subscription %v: not found", subscriptionID)
	}
	localSubscriptions.Delete(subscriptionID)
	return s.Close(subscriptionID, nil)
}

func (s *Subscriptions) SubscribeForEvents(resourceProjection *Projection, srv pb.GrpcGateway_SubscribeForEventsServer) error {
	owner, err := kitNetGrpc.OwnerFromMD(srv.Context())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	var localSubscriptions sync.Map
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

	var sendMutex sync.Mutex
	send := func(e *pb.Event) error {
		log.Debugf("subscriptions.SubscribeForEvents.send: %v %+v", e.GetSubscriptionId(), e.GetType())
		sendMutex.Lock()
		defer sendMutex.Unlock()
		return srv.Send(e)
	}

	for {
		subReq, err := srv.Recv()
		if err == io.EOF {
			log.Debugf("subscriptions.SubscribeForEvents: cannot receive events for owner %v: %v", owner, err)
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err)
		}

		subRes := pb.Event{
			Token: subReq.Token,
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
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
			send(&subRes)
			continue
		}

		subID, err := uuid.NewV4()
		if err != nil {
			subRes.GetOperationProcessed().ErrorStatus.Code = pb.Event_OperationProcessed_ErrorStatus_ERROR
			subRes.GetOperationProcessed().ErrorStatus.Message = fmt.Sprintf("cannot generate subscription ID: %v", err)
			send(&subRes)
			continue
		}

		subRes.SubscriptionId = subID.String()
		localSubscriptions.Store(subRes.SubscriptionId, true)
		send(&subRes)

		switch r := subReq.GetFilterBy().(type) {
		case *pb.SubscribeForEvents_DevicesEvent:
			err = s.SubscribeForDevicesEvent(ctx, owner, resourceProjection, subRes.SubscriptionId, subRes.GetToken(), send, r.DevicesEvent)
		case *pb.SubscribeForEvents_DeviceEvent:
			err = s.SubscribeForDeviceEvent(ctx, owner, resourceProjection, subRes.SubscriptionId, subRes.GetToken(), send, r.DeviceEvent)
		case *pb.SubscribeForEvents_ResourceEvent:
			err = s.SubscribeForResourceEvent(ctx, owner, resourceProjection, subRes.SubscriptionId, subRes.GetToken(), send, r.ResourceEvent)
		case *pb.SubscribeForEvents_CancelSubscription_:
			//handled by cancelation
			err = nil
		default:
			err = fmt.Errorf("not supported")
			send(&pb.Event{
				SubscriptionId: subRes.SubscriptionId,
				Token:          subReq.Token,
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

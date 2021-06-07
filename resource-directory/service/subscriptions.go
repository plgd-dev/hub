package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc/codes"

	"github.com/gofrs/uuid"
	clientAS "github.com/plgd-dev/cloud/authorization/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"

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

	rwlock           sync.RWMutex
	allSubscriptions map[string]Subscriber               // map[subscriptionID]
	subscriptions    map[string]map[string]*subscription // map[userId]map[subscriptionID]

	initSubscriptionsLock sync.Mutex
	initSubscriptions     map[string]map[string]Subscriber // map[userId]map[subscriptionID]
}

type SendEventFunc func(e *pb.Event) error

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		allSubscriptions:  make(map[string]Subscriber),
		subscriptions:     make(map[string]map[string]*subscription),
		initSubscriptions: make(map[string]map[string]Subscriber),
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

func (s *Subscriptions) getSubscriptionsToUpdate(owner string, init map[string]Subscriber) []*subscription {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()
	updated := make([]*subscription, 0, 32)
	for _, sub := range s.subscriptions[owner] {
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
		switch sub.(type) {
		case *subscription:
			delete(s.subscriptions[owner], id)
			if len(s.subscriptions[owner]) == 0 {
				delete(s.subscriptions, owner)
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

func (s *Subscriptions) Insertsubscription(ctx context.Context, sub *subscription) error {
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
	userSubs, ok := s.subscriptions[owner]
	if !ok {
		userSubs = make(map[string]*subscription)
		s.subscriptions[owner] = userSubs
	}
	userSubs[sub.ID()] = sub

	s.insertToInitSubscriptions(sub)
	s.allSubscriptions[sub.ID()] = sub
	return nil
}

func (s *Subscriptions) OnResourceLinksPublished(ctx context.Context, links ResourceLinksPublished) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, links.data.GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
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

func (s *Subscriptions) OnResourceLinksUnpublished(ctx context.Context, event *events.ResourceLinksUnpublished) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, event.GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfUnpublishedResourceLinks(ctx, event); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource unpublished event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceUpdatePending(ctx context.Context, updatePending *events.ResourceUpdatePending) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, updatePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfUpdatePendingResource(ctx, updatePending); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource update pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceUpdated(ctx context.Context, updated *events.ResourceUpdated) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, updated.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfUpdatedResource(ctx, updated); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource updated event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceRetrievePending(ctx context.Context, retrievePending *events.ResourceRetrievePending) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, retrievePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfRetrievePendingResource(ctx, retrievePending); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource retrieve pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceDeletePending(ctx context.Context, deletePending *events.ResourceDeletePending) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deletePending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfDeletePendingResource(ctx, deletePending); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource delete pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceCreatePending(ctx context.Context, createPending *events.ResourceCreatePending) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, createPending.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfCreatePendingResource(ctx, createPending); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource create pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceRetrieved(ctx context.Context, retrieved *events.ResourceRetrieved) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, retrieved.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfRetrievedResource(ctx, retrieved); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource retrieved event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceDeleted(ctx context.Context, deleted *events.ResourceDeleted) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deleted.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfDeletedResource(ctx, deleted); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource deleted event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceCreated(ctx context.Context, created *events.ResourceCreated) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, created.GetResourceId().GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfCreatedResource(ctx, created); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource created event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnResourceContentChanged(ctx context.Context, resourceChanged *events.ResourceChanged) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	deviceID := resourceChanged.GetResourceId().GetDeviceId()
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, deviceID) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfResourceChanged(ctx, resourceChanged); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send resource content changed: %v", errors)
	}
	return nil
}

func (s *Subscriptions) SubscribeForDevicesEvent(ctx context.Context, owner string, resourceProjection *Projection, subscriptionID, token string, send SendEventFunc, req *pb.SubscribeToEvents_CreateSubscription) error {
	sub := Newsubscription(subscriptionID, owner, token, send, resourceProjection, req)
	err := s.Insertsubscription(ctx, sub)
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

func (s *Subscriptions) SubscribeToEvents(resourceProjection *Projection, srv pb.GrpcGateway_SubscribeToEventsServer) error {
	owner, err := kitNetGrpc.OwnerFromMD(srv.Context())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	var localSubscriptions sync.Map
	ctx := srv.Context()

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
		log.Debugf("subscriptions.SubscribeToEvents.send: %v %+v", e.GetSubscriptionId(), e.GetType())
		sendMutex.Lock()
		defer sendMutex.Unlock()
		return srv.Send(e)
	}

	for {
		subReq, err := srv.Recv()
		if err == io.EOF {
			log.Debugf("subscriptions.SubscribeToEvents: cannot receive events for owner %v: %v", owner, err)
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

		switch r := subReq.GetAction().(type) {
		case *pb.SubscribeToEvents_CreateSubscription_:
			err = s.SubscribeForDevicesEvent(ctx, owner, resourceProjection, subRes.SubscriptionId, subRes.GetToken(), send, r.CreateSubscription)
		case *pb.SubscribeToEvents_CancelSubscription_:
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
			log.Errorf("errors occurs during %T: %v", subReq.GetAction(), err)
		}
	}
	return nil
}

func (s *Subscriptions) OnDeviceMetadataUpdatePending(ctx context.Context, event *events.DeviceMetadataUpdatePending) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, event.GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfUpdatePendingDeviceMetadata(ctx, event); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device metadata update pending event: %v", errors)
	}
	return nil
}

func (s *Subscriptions) OnDeviceMetadataUpdated(ctx context.Context, event *events.DeviceMetadataUpdated) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	var errors []error
	for owner, userSubs := range s.subscriptions {
		if !s.userDevicesManager.IsUserDevice(owner, event.GetDeviceId()) {
			continue
		}
		for _, sub := range userSubs {
			if err := sub.NotifyOfUpdatedDeviceMetadata(ctx, event); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot send device metadata updated event: %v", errors)
	}
	return nil
}

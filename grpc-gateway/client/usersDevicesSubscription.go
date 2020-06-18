package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

type devSub struct {
	*DeviceSubscription
}

type UsersDevicesSubscription struct {
	mutex    sync.Mutex
	subs     map[string]map[string]*devSub
	gwClient pb.GrpcGatewayClient
}

func (s *UsersDevicesSubscription) getOrCreate(userID string, deviceID string) (*devSub, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subs, ok := s.subs[userID]
	if !ok {
		subs = make(map[string]*devSub)
		s.subs[userID] = subs
	}
	sub, ok := subs[deviceID]
	if !ok {
		sub = &devSub{}
		subs[deviceID] = sub
	}
	return sub, !ok
}

func (s *UsersDevicesSubscription) get(userID string, deviceID string) *devSub {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subs, ok := s.subs[userID]
	if !ok {
		return nil
	}
	sub, ok := subs[deviceID]
	if !ok {
		return nil
	}
	return sub
}

func (s *UsersDevicesSubscription) pop(userID string, deviceID string) *devSub {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v := s.subs[userID][deviceID]
	delete(s.subs[userID], deviceID)
	if len(s.subs[userID]) == 0 {
		delete(s.subs, userID)
	}
	return v
}

func NewUsersDevicesSubscription(gwClient pb.GrpcGatewayClient) *UsersDevicesSubscription {
	return &UsersDevicesSubscription{
		subs:     make(map[string]map[string]*devSub),
		gwClient: gwClient,
	}
}

func (s *UsersDevicesSubscription) Create(ctx context.Context, userID string, deviceID string, handle SubscriptionHandler) (created bool, err error) {
	devSub, ok := s.getOrCreate(userID, deviceID)
	if !ok {
		return false, nil
	}
	h := NewCloseErrorHandler(
		func() {
			s.pop(userID, deviceID)
			handle.OnClose()
		},
		func(err error) {
			s.pop(userID, deviceID)
			handle.Error(err)
		},
	)
	sub, err := NewDeviceSubscription(kitNetGrpc.CtxWithUserID(ctx, userID), deviceID, h, handle, s.gwClient)
	if err != nil {
		s.pop(userID, deviceID)
		return false, err
	}
	devSub.DeviceSubscription = sub
	return true, nil
}

func (s *UsersDevicesSubscription) Cancel(userID string, deviceID string) (wait func(), err error) {
	sub := s.pop(userID, deviceID)
	if sub == nil {
		return nil, fmt.Errorf("not found")
	}
	return sub.Cancel()
}

func (s *UsersDevicesSubscription) popSubs() map[string]map[string]*devSub {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tmp := s.subs
	s.subs = make(map[string]map[string]*devSub)
	return tmp
}

func (s *UsersDevicesSubscription) Close() {
	subs := s.popSubs()
	for _, devs := range subs {
		for _, s := range devs {
			s.Cancel()
		}
	}
}

package service

import (
	"context"
	"sync"

	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

type UpdateSubscription struct {
	mutex sync.Mutex
	sub   *client.DeviceSubscription
}

func (s *UpdateSubscription) Init(ctx context.Context, userID string, deviceID string, gwClient pb.GrpcGatewayClient) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if sub != nil {
		return
	}
	sub, err := client.NewDeviceSubscription(kitNetGrpc.CtxWithUserID(ctx, userID), deviceID, s.UpdatePending, gwClient)
	if err != nil {
		return err
	}
	s.sub = sub
	return nil
}

type UserUpdateSubscriptions struct {
	mutex    sync.Mutex
	subs     map[string]map[string]*UpdateSubscription
	gwClient pb.GrpcGatewayClient
}

func (s *UserUpdateSubscriptions) getOrCreate(userID string, deviceID string) *UpdateSubscription {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	subs, ok := s.subs[userID]
	if !ok {
		subs = make(map[string]*UpdateSubscription)
		s.subs[userID] = subs
	}
	sub, ok := subs[deviceID]
	if !ok {
		sub = &UpdateSubscription{}
	}
	return sub
}

func (s *UserUpdateSubscriptions) Insert(ctx context.Context, userID string, deviceID string) error {
	sub := getOrCreate(ctx, userID, deviceID)
	err := sub.Init()
	if err != nil {

	}

}

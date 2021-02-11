package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
)

type subscription struct {
	id     string
	userID string
	token  string

	resourceProjection *Projection
	send               SendEventFunc

	lock                          sync.Mutex
	registeredDevicesInProjection map[string]bool

	eventVersionsLock sync.Mutex
	eventVersions     map[string]uint64
}

func NewSubscription(userID, id, token string, send SendEventFunc, resourceProjection *Projection) *subscription {
	return &subscription{
		userID:                        userID,
		id:                            id,
		token:                         token,
		send:                          send,
		resourceProjection:            resourceProjection,
		eventVersions:                 make(map[string]uint64),
		registeredDevicesInProjection: make(map[string]bool),
	}
}

func (s *subscription) UserID() string {
	return s.userID
}

func (s *subscription) ID() string {
	return s.id
}

func (s *subscription) Token() string {
	return s.token
}

func (s *subscription) FilterByVersion(deviceID, href, typeEvent string, version uint64) bool {
	resourceID := utils.MakeResourceID(deviceID, href+"."+typeEvent)

	s.eventVersionsLock.Lock()
	defer s.eventVersionsLock.Unlock()
	v, ok := s.eventVersions[resourceID]
	if !ok {
		s.eventVersions[resourceID] = version
		return false
	}
	if v >= version {
		return true
	}
	s.eventVersions[resourceID] = version
	return false
}

func (s *subscription) RegisterToProjection(ctx context.Context, deviceID string) (loaded bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	loaded, err = s.resourceProjection.Register(ctx, deviceID)
	if err != nil {
		return loaded, err
	}
	s.registeredDevicesInProjection[deviceID] = true
	return
}

func (s *subscription) UnregisterFromProjection(ctx context.Context, deviceID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.registeredDevicesInProjection[deviceID]
	if !ok {
		return nil
	}
	delete(s.registeredDevicesInProjection, deviceID)
	return s.resourceProjection.Unregister(deviceID)
}

func (s *subscription) Send(event *pb.Event) error {
	return s.send(event)
}

func (s *subscription) Close(reason error) error {
	r := ""
	if reason != nil {
		r = reason.Error()
	}

	var errors []error

	err := s.unregisterProjections()
	if err != nil {
		errors = append(errors, err)
	}

	err = s.Send(&pb.Event{
		Token:          s.Token(),
		SubscriptionId: s.ID(),
		Type: &pb.Event_SubscriptionCanceled_{
			SubscriptionCanceled: &pb.Event_SubscriptionCanceled{
				Reason: r,
			},
		},
	})
	if err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot close subscription %v: %v", s.ID(), errors)
	}

	return nil
}

func (s *subscription) unregisterProjections() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var errors []error
	for deviceID := range s.registeredDevicesInProjection {
		err := s.resourceProjection.Unregister(deviceID)
		if err != nil {
			errors = append(errors, err)
		}
		delete(s.registeredDevicesInProjection, deviceID)
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot unregister projections for %v: %v", s.ID(), errors)
	}
	return nil
}

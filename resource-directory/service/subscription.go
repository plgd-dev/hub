package service

import (
	"context"
	"fmt"
	"hash/crc64"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
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
	eventVersions     map[string]subscriptionEvent
}

type subscriptionEvent struct {
	version uint64
	hash    uint64
}

func NewSubscription(userID, id, token string, send SendEventFunc, resourceProjection *Projection) *subscription {
	return &subscription{
		userID:                        userID,
		id:                            id,
		token:                         token,
		send:                          send,
		resourceProjection:            resourceProjection,
		eventVersions:                 make(map[string]subscriptionEvent),
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

func CalcHashFromBytes(content []byte) uint64 {
	if len(content) > 0 {
		return crc64.Checksum(content, crc64.MakeTable(crc64.ISO))
	}
	return 0
}

func (s *subscription) FilterByVersionAndHash(deviceID, href, typeEvent string, version, hash uint64) bool {
	resourceID := (&commands.ResourceId{DeviceId: deviceID, Href: href + "." + typeEvent}).ToUUID()
	s.eventVersionsLock.Lock()
	defer s.eventVersionsLock.Unlock()
	v, ok := s.eventVersions[resourceID]
	if !ok {
		s.eventVersions[resourceID] = subscriptionEvent{
			version: version,
			hash:    hash,
		}
		return false
	}
	if v.version >= version {
		return true
	}
	var ret bool
	if hash != 0 {
		ret = v.hash == hash
		v.hash = hash
	}
	v.version = version
	s.eventVersions[resourceID] = v
	return ret
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

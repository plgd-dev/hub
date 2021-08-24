package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/authorization/events"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/strings"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	kitSync "github.com/plgd-dev/kit/sync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ownerSubject struct {
	handlers      map[uint64]func(e *events.Event)
	subscription  *nats.Subscription
	devices       strings.SortedSlice
	validUntil    time.Time
	devicesSynced bool
	sync.Mutex
}

func newOwnerSubject(validUntil time.Time) *ownerSubject {
	return &ownerSubject{
		handlers:   make(map[uint64]func(e *events.Event)),
		devices:    make(strings.SortedSlice, 0, 16),
		validUntil: validUntil,
	}
}

func (d *ownerSubject) Handle(msg *nats.Msg) error {
	var e events.Event
	if err := utils.Unmarshal(msg.Data, &e); err != nil {
		return err
	}
	d.Lock()
	if d.devicesSynced {
		d.devices = d.devices.Insert(e.GetDevicesRegistered().GetDeviceIds()...)
		d.devices = d.devices.Remove(e.GetDevicesUnregistered().GetDeviceIds()...)
	}
	handlers := make(map[uint64]func(e *events.Event))
	for key, h := range d.handlers {
		handlers[key] = h
	}
	d.Unlock()
	for _, h := range handlers {
		h(&e)
	}

	return nil
}

func (d *ownerSubject) AddHandlerLocked(id uint64, h func(e *events.Event)) bool {
	if _, ok := d.handlers[id]; !ok {
		d.handlers[id] = h
		return true
	}
	return false
}

func (d *ownerSubject) RemoveHandlerLocked(v uint64) {
	delete(d.handlers, v)
}

func (d *ownerSubject) updateDevicesLocked(devices []string) ([]string, []string) {
	deviceIDs := strings.MakeSortedSlice(devices)
	added := deviceIDs.Difference(d.devices)
	removed := d.devices.Difference(deviceIDs)
	d.devices = deviceIDs
	d.devicesSynced = true
	return added, removed
}

func (d *ownerSubject) subscribeLocked(owner string, subscribe func(subj string, cb nats.MsgHandler) (*nats.Subscription, error), handle func(msg *nats.Msg)) error {
	if d.subscription == nil {
		sub, err := subscribe(events.GetOwnerSubject(owner), handle)
		if err != nil {
			return err
		}
		d.subscription = sub
	}
	return nil
}

type OwnerCache struct {
	owners     *kitSync.Map
	conn       *nats.Conn
	ownerClaim string
	errFunc    ErrFunc
	asClient   pbAS.AuthorizationServiceClient
	expiration time.Duration
	handlerID  uint64

	done chan struct{}
	wg   sync.WaitGroup
}

func NewOwnerCache(ownerClaim string, expiration time.Duration, conn *nats.Conn, asClient pbAS.AuthorizationServiceClient, errFunc ErrFunc) *OwnerCache {
	c := &OwnerCache{
		owners:     kitSync.NewMap(),
		conn:       conn,
		ownerClaim: ownerClaim,
		errFunc:    errFunc,
		asClient:   asClient,
		expiration: expiration,
		done:       make(chan struct{}),
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run()
	}()
	return c
}

func (c *OwnerCache) makeCloseFunc(owner string, id uint64) func() {
	return func() {
		c.owners.ReplaceWithFunc(owner, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
			if !oldLoaded {
				return nil, true
			}
			s := oldValue.(*ownerSubject)
			s.Lock()
			defer s.Unlock()
			s.RemoveHandlerLocked(id)
			now := time.Now()
			if len(s.handlers) == 0 && s.validUntil.After(now) {
				s.validUntil = now.Add(c.expiration)
			}
			return s, false
		})
	}
}

func (c *OwnerCache) getOwnerDevices(ctx context.Context, asClient pbAS.AuthorizationServiceClient) ([]string, error) {
	getUserDevicesClient, err := asClient.GetUserDevices(ctx, &pbAS.GetUserDevicesRequest{})
	if err != nil {
		return nil, status.Errorf(status.Convert(err).Code(), "cannot get owners devices: %v", err)
	}
	defer func() {
		if err := getUserDevicesClient.CloseSend(); err != nil {
			c.errFunc(fmt.Errorf("cannot close send direction of get owners devices stream: %v", err))
		}
	}()
	ownerDevices := make([]string, 0, 32)
	for {
		userDevice, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(status.Convert(err).Code(), "cannot get owners devices: %v", err)
		}
		ownerDevices = append(ownerDevices, userDevice.DeviceId)
	}
	return ownerDevices, nil
}

func (c *OwnerCache) getOrSetLockedOwnerSubject(owner string) *ownerSubject {
	val, _ := c.owners.LoadOrStoreWithFunc(owner, func(value interface{}) interface{} {
		v := value.(*ownerSubject)
		v.Lock()
		return v
	}, func() interface{} {
		v := newOwnerSubject(time.Now().Add(c.expiration))
		v.Lock()
		return v
	})
	return val.(*ownerSubject)
}

// Subscribe register onEvents handler and creates NATS subscription, if not exists.
// To free subscription call close function.
func (c *OwnerCache) Subscribe(owner string, onEvent func(e *events.Event)) (close func(), err error) {
	s := c.getOrSetLockedOwnerSubject(owner)

	closeFunc := func() {}
	if onEvent != nil {
		handlerId := atomic.AddUint64(&c.handlerID, 1)
		for !s.AddHandlerLocked(handlerId, onEvent) {
			handlerId = atomic.AddUint64(&c.handlerID, 1)
		}
		closeFunc = c.makeCloseFunc(owner, handlerId)
	}
	if s.subscription == nil {
		err := s.subscribeLocked(owner, c.conn.Subscribe, func(msg *nats.Msg) {
			if err := s.Handle(msg); err != nil {
				c.errFunc(err)
			}
		})
		if err != nil {
			s.Unlock()
			closeFunc()
			return nil, err
		}
	}
	s.Unlock()
	return closeFunc, nil
}

// Update updates devices in cache and subscribe to NATS for updating them.
func (c *OwnerCache) Update(ctx context.Context) (added []string, removed []string, err error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, c.ownerClaim)
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	s := c.getOrSetLockedOwnerSubject(owner)
	defer s.Unlock()
	err = s.subscribeLocked(owner, c.conn.Subscribe, func(msg *nats.Msg) {
		if err := s.Handle(msg); err != nil {
			c.errFunc(err)
		}
	})
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	now := time.Now()
	devices, err := c.getOwnerDevices(ctx, c.asClient)
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	added, removed = s.updateDevicesLocked(devices)
	s.validUntil = now.Add(c.expiration)
	return added, removed, nil
}

// Returns locked, non-expired owner subject from cache
func (c *OwnerCache) getLockedOwnerSubject(owner string) (*ownerSubject, bool) {
	var s *ownerSubject
	now := time.Now()
	c.owners.LoadWithFunc(owner, func(value interface{}) interface{} {
		s = value.(*ownerSubject)
		s.Lock()
		return s
	})
	if s == nil {
		return nil, false
	}
	if !s.devicesSynced {
		s.Unlock()
		return nil, false
	}
	s.validUntil = now.Add(c.expiration)
	return s, true
}

// GetDevices provides the owner of the cached device. If the cache does not expire, the cache expiration is extended.
// When ok == false you need to Update to refresh cache.
func (c *OwnerCache) GetDevices(owner string) (devices []string, ok bool) {
	s, ok := c.getLockedOwnerSubject(owner)
	if !ok {
		return nil, false
	}
	defer s.Unlock()

	devices = make([]string, len(s.devices))
	copy(devices, s.devices)
	return devices, true
}

type CacheResult int

const (
	NeedsUpdate = 0
	NotFound    = 1
	Found       = 2
)

func (c *OwnerCache) OwnsDevice(owner, deviceID string) CacheResult {
	s, ok := c.getLockedOwnerSubject(owner)
	if !ok {
		return NeedsUpdate
	}
	defer s.Unlock()
	if s.devices.Contains(deviceID) {
		return Found
	}
	return NotFound
}

func (c *OwnerCache) getExpiredOwnerSubjects(t time.Time) []string {
	expiredOwners := make([]string, 0, 32)
	c.owners.Range(func(key, value interface{}) bool {
		s := value.(*ownerSubject)
		s.Lock()
		defer s.Unlock()
		if len(s.handlers) > 0 {
			if s.validUntil.Before(t) {
				//expire devices in cache - user needs to call UpdateDevices to refresh them
				s.devices = s.devices[:0]
				s.devicesSynced = false
			}
			return true
		}
		if s.validUntil.Before(t) {
			expiredOwners = append(expiredOwners, key.(string))
		}
		return true
	})
	return expiredOwners
}

func (c *OwnerCache) checkExpiration() {
	now := time.Now()
	expiredOwners := c.getExpiredOwnerSubjects(now)

	for _, o := range expiredOwners {
		var unsubscribeSubscription *nats.Subscription
		c.owners.ReplaceWithFunc(o, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
			if !oldLoaded {
				return nil, true
			}
			s := oldValue.(*ownerSubject)
			s.Lock()
			defer s.Unlock()
			if len(s.handlers) > 0 {
				return s, false
			}
			if s.validUntil.Before(now) {
				unsubscribeSubscription = s.subscription
				return nil, true
			}
			return s, false
		})
		if unsubscribeSubscription != nil {
			if err := unsubscribeSubscription.Unsubscribe(); err != nil {
				c.errFunc(fmt.Errorf("cannot unsubscribe owner('%v'): %w", o, err))
			}
		}
	}
}

func (c *OwnerCache) run() {
	ticker := time.NewTicker(c.expiration / 2)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.checkExpiration()
		}
	}
}

func (c *OwnerCache) Close() {
	close(c.done)
	c.wg.Wait()
}

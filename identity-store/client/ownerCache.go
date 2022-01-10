package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	kitSync "github.com/plgd-dev/kit/v2/sync"
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
		sub, err := subscribe(events.GetRegistrationSubject(owner), handle)
		if err != nil {
			return err
		}
		d.subscription = sub
	}
	return nil
}

func (d *ownerSubject) syncDevicesLocked(ctx context.Context, owner string, cache *OwnerCache) (added []string, removed []string, err error) {
	err = d.subscribeLocked(owner, cache.conn.Subscribe, func(msg *nats.Msg) {
		if err := d.Handle(msg); err != nil {
			cache.errFunc(err)
		}
	})
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	now := time.Now()
	devices, err := cache.getOwnerDevices(ctx, cache.isClient)
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	d.validUntil = now.Add(cache.expiration)
	added, removed = d.updateDevicesLocked(devices)
	return added, removed, nil
}

// ErrFunc reports errors
type ErrFunc = func(err error)

type OwnerCache struct {
	owners     *kitSync.Map
	conn       *nats.Conn
	ownerClaim string
	errFunc    ErrFunc
	isClient   pbIS.IdentityStoreClient
	expiration time.Duration
	handlerID  uint64

	done chan struct{}
	wg   sync.WaitGroup
}

func NewOwnerCache(ownerClaim string, expiration time.Duration, conn *nats.Conn, isClient pbIS.IdentityStoreClient, errFunc ErrFunc) *OwnerCache {
	c := &OwnerCache{
		owners:     kitSync.NewMap(),
		conn:       conn,
		ownerClaim: ownerClaim,
		errFunc:    errFunc,
		isClient:   isClient,
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

func (c *OwnerCache) OwnerClaim() string {
	return c.ownerClaim
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

func (c *OwnerCache) getOwnerDevices(ctx context.Context, isClient pbIS.IdentityStoreClient) ([]string, error) {
	getDevicesClient, err := isClient.GetDevices(ctx, &pbIS.GetDevicesRequest{})
	if err != nil {
		return nil, status.Errorf(status.Convert(err).Code(), "cannot get owners devices: %v", err)
	}
	defer func() {
		if err := getDevicesClient.CloseSend(); err != nil {
			c.errFunc(fmt.Errorf("cannot close send direction of get owners devices stream: %v", err))
		}
	}()
	ownerDevices := make([]string, 0, 32)
	for {
		device, err := getDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(status.Convert(err).Code(), "cannot receive owners devices: %v", err)
		}
		ownerDevices = append(ownerDevices, device.DeviceId)
	}
	return ownerDevices, nil
}

// Create or get owner subject, lock it, execute function and unlock it
func (c *OwnerCache) executeOnLockedOwnerSubject(owner string, fn func(*ownerSubject) error) error {
	val, _ := c.owners.LoadOrStoreWithFunc(owner, func(value interface{}) interface{} {
		v := value.(*ownerSubject)
		v.Lock()
		return v
	}, func() interface{} {
		v := newOwnerSubject(time.Now().Add(c.expiration))
		v.Lock()
		return v
	})
	s := val.(*ownerSubject)
	defer s.Unlock()
	return fn(s)
}

// Subscribe register onEvents handler and creates a NATS subscription, if it does not exist.
// To free subscription call the returned close function.
func (c *OwnerCache) Subscribe(owner string, onEvent func(e *events.Event)) (close func(), err error) {
	closeFunc := func() {
		// Do nothing if no owner subject is found
	}
	err = c.executeOnLockedOwnerSubject(owner, func(s *ownerSubject) error {
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
				return err
			}
		}
		return nil
	})
	if err != nil {
		closeFunc()
		return nil, err
	}

	return closeFunc, nil
}

// Update updates devices in cache and subscribe to NATS for updating them.
func (c *OwnerCache) Update(ctx context.Context) (added []string, removed []string, err error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, c.ownerClaim)
	if err != nil {
		return nil, nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	err = c.executeOnLockedOwnerSubject(owner, func(s *ownerSubject) error {
		added, removed, err = s.syncDevicesLocked(ctx, owner, c)
		return err
	})

	if err != nil {
		return nil, nil, err
	}

	return added, removed, nil
}

// GetDevices provides the owner of the cached device. If the cache does not expire, the cache expiration is extended.
func (c *OwnerCache) GetDevices(ctx context.Context) (devices []string, err error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, c.ownerClaim)
	if err != nil {
		return nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	now := time.Now()
	if err = c.executeOnLockedOwnerSubject(owner, func(s *ownerSubject) error {
		if !s.devicesSynced {
			if _, _, err := s.syncDevicesLocked(ctx, owner, c); err != nil {
				return err
			}
		} else {
			s.validUntil = now.Add(c.expiration)
		}
		devices = make([]string, len(s.devices))
		copy(devices, s.devices)
		return nil
	}); err != nil {
		return nil, err
	}

	return devices, nil
}

// Check provided list of device ids and return only ids owned by the user
func (c *OwnerCache) GetSelectedDevices(ctx context.Context, devices []string) ([]string, error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, c.ownerClaim)
	if err != nil {
		return nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	deviceIds := strings.MakeSortedSlice(devices)
	if err = c.executeOnLockedOwnerSubject(owner, func(s *ownerSubject) error {
		if !s.devicesSynced {
			if _, _, err := s.syncDevicesLocked(ctx, owner, c); err != nil {
				return err
			}
		}
		deviceIds = s.devices.Intersection(deviceIds)
		return nil
	}); err != nil {
		return nil, err
	}

	return deviceIds, nil
}

// Check if all provided devices are owned by the user
func (c *OwnerCache) OwnsDevices(ctx context.Context, devices []string) (bool, error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, c.ownerClaim)
	if err != nil {
		return false, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}

	equalDevices := false
	deviceIds := strings.MakeSortedSlice(devices)
	if err = c.executeOnLockedOwnerSubject(owner, func(s *ownerSubject) error {
		if !s.devicesSynced {
			if _, _, err := s.syncDevicesLocked(ctx, owner, c); err != nil {
				return err
			}
		}
		equalDevices = s.devices.IsSuperslice(deviceIds)
		return nil
	}); err != nil {
		return false, err
	}

	return equalDevices, nil
}

// Convenience method to check if given device is owned by the user
func (c *OwnerCache) OwnsDevice(ctx context.Context, deviceID string) (bool, error) {
	return c.OwnsDevices(ctx, []string{deviceID})
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

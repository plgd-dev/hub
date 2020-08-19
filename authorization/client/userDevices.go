package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/patrickmn/go-cache"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	kitSync "github.com/plgd-dev/kit/sync"
	"google.golang.org/grpc/status"
)

// UserDevicesManager provides notification mechanism about devices.
type UserDevicesManager struct {
	fn       TriggerFunc
	asClient pbAS.AuthorizationServiceClient
	errFunc  ErrFunc

	lock                sync.RWMutex
	users               map[string]*kitSync.RefCounter
	done                context.CancelFunc
	trigger             chan triggerUserDevice
	doneWg              sync.WaitGroup
	getUserDevicesCache *cache.Cache
}

// TriggerFunc notifies users remove/add device.
type TriggerFunc = func(ctx context.Context, userID string, addedDevices, removedDevices, currentDevices map[string]bool)

// ErrFunc reports errors
type ErrFunc func(err error)

// NewUserDevicesManager creates userID devices manager.
func NewUserDevicesManager(fn TriggerFunc, asClient pbAS.AuthorizationServiceClient, tickFrequency, expiration time.Duration, errFunc ErrFunc) *UserDevicesManager {
	c := cache.New(expiration, cache.DefaultExpiration)
	c.OnEvicted(func(key string, v interface{}) {
		r := v.(*kitSync.RefCounter)
		r.Release(context.Background())
	})

	ctx, cancel := context.WithCancel(context.Background())

	m := &UserDevicesManager{
		fn:                  fn,
		asClient:            asClient,
		done:                cancel,
		trigger:             make(chan triggerUserDevice, 32),
		users:               make(map[string]*kitSync.RefCounter),
		errFunc:             errFunc,
		getUserDevicesCache: c,
	}
	m.doneWg.Add(1)
	go m.run(ctx, tickFrequency, expiration)
	return m
}

func (d *UserDevicesManager) getRef(userID string, update bool) (_ *kitSync.RefCounter, created bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	u, ok := d.users[userID]
	created = false
	if !ok && update {
		u = kitSync.NewRefCounter(&userDevices{
			devices: make(map[string]bool),
			userID:  userID,
			lock:    semaphore.NewWeighted(1),
			validTo: time.Now().Add(time.Hour * 24),
		}, func(ctx context.Context, data interface{}) error {
			u := data.(*userDevices)
			d.fn(ctx, u.userID, nil, u.getDevices(), nil)
			return nil
		})
		d.users[userID] = u
		created = true
	} else if ok {
		u.Acquire()
	}
	return u, created
}

// Acquire acquires reference counter by 1 for userID.
func (d *UserDevicesManager) Acquire(ctx context.Context, userID string) error {
	d.getRef(userID, true)
	userDevices, err := getUsersDevices(ctx, d.asClient, []string{userID})
	if err != nil {
		d.Release(userID)
		return err
	}
	d.trigger <- triggerUserDevice{
		userID:      userID,
		userDevices: userDevices,
		update:      true,
	}
	return nil
}

// GetUserDevices returns devices which belows to user.
func (d *UserDevicesManager) GetUserDevices(ctx context.Context, userID string) ([]string, error) {
	v, created := d.getRef(userID, true)
	if created {
		userDevices, err := getUsersDevices(ctx, d.asClient, []string{userID})
		if err != nil {
			d.Release(userID)
			return nil, err
		}
		d.trigger <- triggerUserDevice{
			userID:      userID,
			userDevices: userDevices,
			update:      true,
		}
		d.getUserDevicesCache.Add(userID, v, cache.DefaultExpiration)
		return userDevices[userID], nil
	}
	defer d.Release(userID) // getRef increase ref counter
	mapDevs := v.Data().(*userDevices).getDevices()
	devs := make([]string, 0, len(mapDevs))
	for d := range mapDevs {
		devs = append(devs, d)
	}
	return devs, nil
}

func (d *UserDevicesManager) IsUserDevice(userID, deviceID string) bool {
	v, _ := d.getRef(userID, false)
	if v == nil {
		return false
	}
	defer d.Release(userID) // getRef increase ref counter
	return v.Data().(*userDevices).isUserDevice(deviceID)
}

// Release releases reference counter by 1 over userID.
func (d *UserDevicesManager) release(userID string) *kitSync.RefCounter {
	d.lock.Lock()
	defer d.lock.Unlock()
	u, ok := d.users[userID]
	if !ok {
		return nil
	}
	if 1 == u.Count() {
		delete(d.users, userID)
	}
	return u
}

// Release releases reference counter by 1 over userID.
func (d *UserDevicesManager) Release(userID string) error {
	u := d.release(userID)
	if u != nil {
		return u.Release(context.Background())
	}
	return nil
}

func (d *UserDevicesManager) updateDevices(ctx context.Context, userID string, deviceIDs []string, validTo time.Time) (added, removed, allDevices map[string]bool) {
	v, _ := d.getRef(userID, false)
	if v == nil {
		return
	}
	defer func() {
		err := d.Release(userID)
		if err != nil {
			d.errFunc(fmt.Errorf("cannot release userID %v devices: %w", userID, err))
		}
	}()

	return v.Data().(*userDevices).updateDevices(deviceIDs, validTo)
}

func (d *UserDevicesManager) getDevices(ctx context.Context, userID string) map[string]bool {
	v, _ := d.getRef(userID, false)
	if v == nil {
		return nil
	}
	defer func() {
		err := d.Release(userID)
		if err != nil {
			d.errFunc(fmt.Errorf("cannot release userID %v devices: %w", userID, err))
		}
	}()
	return v.Data().(*userDevices).getDevices()
}

func (d *UserDevicesManager) getUsers(triggerTime time.Time) []string {
	d.lock.Lock()
	defer d.lock.Unlock()
	users := make([]string, 0, len(d.users))
	for u, v := range d.users {
		if v.Data().(*userDevices).isExpired(triggerTime) {
			users = append(users, u)
		}
	}
	return users
}

func getUsersDevices(ctx context.Context, asClient pbAS.AuthorizationServiceClient, usedIDs []string) (map[string][]string, error) {
	getUserDevicesClient, err := asClient.GetUserDevices(ctx, &pbAS.GetUserDevicesRequest{
		UserIdsFilter: usedIDs,
	})
	if err != nil {
		return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
	}
	defer getUserDevicesClient.CloseSend()
	userDevices := make(map[string][]string)
	for _, userID := range usedIDs {
		userDevices[userID] = make([]string, 0, 32)
	}
	for {
		userDevice, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
		}
		devices, ok := userDevices[userDevice.UserId]
		if ok {
			userDevices[userDevice.UserId] = append(devices, userDevice.DeviceId)
		}
	}
	return userDevices, nil
}

func (d *UserDevicesManager) onTick(ctx context.Context, timeout time.Duration, expiration time.Duration, triggerTime time.Time) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	users := d.getUsers(triggerTime)
	if len(users) == 0 {
		return
	}
	usersDevices, err := getUsersDevices(ctx, d.asClient, users)
	if err != nil {
		d.errFunc(fmt.Errorf("cannot get user devices: %w", err))
		return
	}
	for userID, devices := range usersDevices {
		added, removed, all := d.updateDevices(ctx, userID, devices, time.Now().Add(expiration))
		if len(added) != 0 || len(removed) != 0 {
			d.fn(ctx, userID, added, removed, all)
		}
	}
}

func (d *UserDevicesManager) onTrigger(ctx context.Context, timeout time.Duration, expiration time.Duration, t triggerUserDevice) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if t.update {
		for userID, devices := range t.userDevices {
			added, removed, all := d.updateDevices(ctx, userID, devices, time.Now().Add(expiration))
			d.fn(ctx, t.userID, added, removed, all)
		}
	} else {
		all := d.getDevices(ctx, t.userID)
		d.fn(ctx, t.userID, nil, nil, all)
	}
}

func (d *UserDevicesManager) run(ctx context.Context, tickFrequency, expiration time.Duration) {
	ticker := time.NewTicker(tickFrequency)
	defer d.doneWg.Done()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case triggerTime := <-ticker.C:
			d.onTick(ctx, tickFrequency, expiration, triggerTime)
		case t := <-d.trigger:
			d.onTrigger(ctx, tickFrequency, expiration, t)
		}
	}
}

// Close stops userID manager goroutine.
func (d *UserDevicesManager) Close() {
	d.done()
	d.doneWg.Wait()
}

type triggerUserDevice struct {
	userID      string
	userDevices map[string][]string
	update      bool
}

type userDevices struct {
	lock    *semaphore.Weighted
	userID  string
	devices map[string]bool
	validTo time.Time
}

func (u *userDevices) isExpired(now time.Time) bool {
	if !u.lock.TryAcquire(1) {
		return false
	}
	defer u.lock.Release(1)
	if u.validTo.Sub(now) <= 0 {
		return true
	}
	return false
}

func (u *userDevices) getDevices() map[string]bool {
	u.lock.Acquire(context.Background(), 1)
	defer u.lock.Release(1)
	devices := make(map[string]bool)
	for deviceID := range u.devices {
		devices[deviceID] = true
	}
	return devices
}

func (u *userDevices) isUserDevice(deviceID string) bool {
	u.lock.Acquire(context.Background(), 1)
	defer u.lock.Release(1)
	return u.devices[deviceID]
}

func (u *userDevices) setDevices(deviceIDs []string) {
	devices := make(map[string]bool)
	for _, deviceID := range deviceIDs {
		devices[deviceID] = true
	}
	u.lock.Acquire(context.Background(), 1)
	defer u.lock.Release(1)
	u.devices = devices
}

func (u *userDevices) updateDevices(deviceIDs []string, validTo time.Time) (added, removed, allDevices map[string]bool) {

	added = make(map[string]bool)
	u.lock.Acquire(context.Background(), 1)
	defer u.lock.Release(1)
	for _, deviceID := range deviceIDs {
		_, ok := u.devices[deviceID]
		if !ok {
			added[deviceID] = true
		}
	}

	removed = make(map[string]bool)
	for deviceID := range u.devices {
		removed[deviceID] = true
	}
	for _, deviceID := range deviceIDs {
		_, ok := removed[deviceID]
		if ok {
			delete(removed, deviceID)
		}
	}

	devices := make(map[string]bool)
	for _, deviceID := range deviceIDs {
		devices[deviceID] = true
	}
	u.devices = devices
	u.validTo = validTo

	allDevices = make(map[string]bool)
	for _, deviceID := range deviceIDs {
		allDevices[deviceID] = true
	}

	return added, removed, allDevices
}

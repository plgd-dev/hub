package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"go.opentelemetry.io/otel/trace"
)

type LinkedHubCache struct {
	expiration  time.Duration
	store       *mongodb.Store
	logger      log.Logger
	fileWatcher *fsnotify.Watcher
	wg          *sync.WaitGroup

	hubs   map[string]*future.Future
	mutex  sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	tracerProvider trace.TracerProvider
}

func NewLinkedHubCache(ctx context.Context, expiration time.Duration, store *mongodb.Store, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) *LinkedHubCache {
	ctx, cancel := context.WithCancel(ctx)
	c := &LinkedHubCache{
		ctx:            ctx,
		cancel:         cancel,
		expiration:     expiration,
		hubs:           make(map[string]*future.Future),
		wg:             new(sync.WaitGroup),
		logger:         logger,
		tracerProvider: tracerProvider,
		fileWatcher:    fileWatcher,
		store:          store,
	}
	c.wg.Add(2)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(expiration)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				fn := c.cleanUpExpiredHubs(now)
				fn.Execute()
			}
		}
	}()
	go func() {
		defer c.wg.Done()
		err := c.watch()
		if err != nil {
			logger.Warnf("cannot watch for DB changes in hubs: %v", err)
		}
	}()
	return c
}

func (c *LinkedHubCache) removeByID(id string) {
	f := c.loadFuture(id)
	if f == nil {
		return
	}
	v, err := f.Get(c.ctx)
	if err != nil {
		return
	}
	h, ok := v.(*LinkedHub)
	if !ok {
		return
	}
	h.Invalidate()
}

func (c *LinkedHubCache) watch() error {
	iter, err := c.store.WatchHubs(c.ctx)
	if err != nil {
		return err
	}
	for {
		_, id, ok := iter.Next(c.ctx)
		if !ok {
			break
		}
		c.removeByID(id)
	}
	err = iter.Err()
	errClose := iter.Close()
	if err == nil {
		err = errClose
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (c *LinkedHubCache) tryToRemoveExpiredHubLocked(key string, f *future.Future, now time.Time, wantToRefresh bool) (func(), bool) {
	if !f.Ready() {
		return nil, false
	}
	v, err := f.Get(context.Background())
	if err != nil {
		delete(c.hubs, key)
		return func() {
			// do nothing
		}, true
	}
	h, ok := v.(*LinkedHub)
	if !ok {
		delete(c.hubs, key)
		return func() {
			// do nothing
		}, true
	}

	if h.IsExpired(now) {
		delete(c.hubs, key)
		return h.Close, true
	}
	if wantToRefresh {
		h.Refresh(now)
	}
	return func() {
		// do nothing
	}, false
}

func (c *LinkedHubCache) cleanUpExpiredHubs(now time.Time) fn.FuncList {
	var closer fn.FuncList
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for id, f := range c.hubs {
		closeHub, ok := c.tryToRemoveExpiredHubLocked(id, f, now, false)
		if ok {
			closer.AddFunc(closeHub)
		}
	}
	return closer
}

func (c *LinkedHubCache) getHubs(ctx context.Context, eg *EnrollmentGroup) ([]*LinkedHub, error) {
	hubs := make(map[string]*pb.Hub, len(eg.GetHubIds())+2)
	err := c.store.LoadHubs(ctx, eg.GetOwner(), &store.HubsQuery{
		HubIdFilter: eg.GetHubIds(),
	}, func(ctx context.Context, iter store.HubIter) (err error) {
		for {
			var cfg pb.Hub
			ok := iter.Next(ctx, &cfg)
			if !ok {
				break
			}
			hubs[cfg.GetHubId()] = &cfg
		}

		return iter.Err()
	})
	if err != nil {
		return nil, err
	}
	linkedHubs := make([]*LinkedHub, 0, len(hubs))
	var errs *multierror.Error
	for _, hubID := range eg.GetHubIds() {
		hub, ok := hubs[hubID]
		if !ok {
			errs = multierror.Append(errs, fmt.Errorf("cannot create linked hub(hubId: %v): not found", hubID))
			continue
		}
		h, err := NewLinkedHub(ctx, c.expiration, hub, c.fileWatcher, c.logger, c.tracerProvider)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("cannot create linked hub(id:%v, hubId: %v): %w", hub.GetId(), hub.GetHubId(), err))
			continue
		}
		linkedHubs = append(linkedHubs, h)
	}
	if len(linkedHubs) == 0 {
		err := errs.ErrorOrNil()
		if err != nil {
			return nil, fmt.Errorf("cannot find any hub with ids('%v'): %w", eg.GetHubIds(), err)
		}
		return nil, fmt.Errorf("cannot find any hub with ids: %v", eg.GetHubIds())
	}
	if errs != nil {
		c.logger.Debugf("some error occurs during load linked hubs: %v", errs.Error())
	}
	return linkedHubs, nil
}

func (c *LinkedHubCache) loadFuture(key string) *future.Future {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	f, ok := c.hubs[key]
	if !ok {
		return nil
	}
	return f
}

func (c *LinkedHubCache) getFutureToken(key string, now time.Time) (*future.Future, future.SetFunc, fn.FuncList) {
	var closer fn.FuncList
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for {
		f, ok := c.hubs[key]
		if !ok {
			fu, set := future.New()
			c.hubs[key] = fu
			return fu, set, closer
		}
		closeHub, ok := c.tryToRemoveExpiredHubLocked(key, f, now, true)
		if !ok {
			return f, nil, closer
		}
		closer.AddFunc(closeHub)
	}
}

func (c *LinkedHubCache) pullOutAll() map[string]*future.Future {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	hubs := c.hubs
	c.hubs = make(map[string]*future.Future)
	return hubs
}

func (c *LinkedHubCache) GetHubs(ctx context.Context, eg *EnrollmentGroup) ([]*LinkedHub, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("deadline is not set in ctx")
	}
	f, set, closer := c.getFutureToken(eg.GetId(), time.Now())
	defer closer.Execute()
	if set == nil {
		v, err := f.Get(ctx)
		if err != nil {
			return nil, err
		}
		h, ok := v.([]*LinkedHub)
		if !ok {
			return nil, fmt.Errorf("invalid object type(%T) in a future", v)
		}
		return h, err
	}
	hubs, err := c.getHubs(ctx, eg)
	set(hubs, err)
	if err != nil {
		return nil, err
	}
	return hubs, nil
}

func (c *LinkedHubCache) Close() {
	c.cancel()
	c.wg.Wait()
	for _, f := range c.pullOutAll() {
		v, err := f.Get(context.Background())
		if err != nil {
			continue
		}
		h, ok := v.(*LinkedHub)
		if !ok {
			continue
		}
		h.Close()
	}
}

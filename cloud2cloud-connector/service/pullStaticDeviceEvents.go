package service

import (
	"context"
	"sync"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/kit/log"
	kitSync "github.com/go-ocf/kit/sync"
	"golang.org/x/sync/semaphore"
)

type pullDevice struct {
	linkedCloud   store.LinkedCloud
	linkedAccount store.LinkedAccount
	deviceID      string
}

type StaticDeviceEvents struct {
	registeredDevices *kitSync.Map //[userid+deviceID]
	pullDeviceChan    chan pullDevice
	wg                *sync.WaitGroup
	poolGets          *semaphore.Weighted
	timeout           time.Duration
	raClient          pbRA.ResourceAggregateClient
}

func NewStaticDeviceEvents(raClient pbRA.ResourceAggregateClient, maxParallelGets int64, cacheSize int, timeout time.Duration) *StaticDeviceEvents {
	return &StaticDeviceEvents{
		registeredDevices: kitSync.NewMap(),
		pullDeviceChan:    make(chan pullDevice, cacheSize),
		wg:                &sync.WaitGroup{},
		poolGets:          semaphore.NewWeighted(maxParallelGets),
		timeout:           timeout,
		raClient:          raClient,
	}
}

func (h *StaticDeviceEvents) Trigger(e pullDevice) {
	h.pullDeviceChan <- e
}

func (h *StaticDeviceEvents) pullDevice(ctx context.Context, e pullDevice, subscriptionManager *SubscriptionManager) {
	key := getKey(e.linkedAccount.UserID, e.deviceID)
	_, loaded := h.registeredDevices.LoadOrStore(key, e)
	if loaded {
		return
	}
	err := h.poolGets.Acquire(ctx, 1)
	if err != nil {
		log.Errorf("cannot acquire go routine from pool for device %v with linked linkedAccount(%v): %v", e.deviceID, e.linkedAccount, err)
		return
	}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer h.poolGets.Release(1)
		ctx, cancel := context.WithTimeout(ctx, h.timeout)
		defer cancel()
		var device RetrieveDeviceWithLinksResponse
		err := Get(ctx, e.linkedCloud.Endpoint.URL+"/devices/"+e.deviceID, e.linkedAccount, e.linkedCloud, &device)
		if err != nil {
			log.Errorf("cannot pull device %v for linked linkedAccount(%v): %v", e.deviceID, e.linkedAccount, err)
			h.registeredDevices.Delete(key)
		}
		err = publishDeviceResources(ctx, h.raClient, e.deviceID, e.linkedAccount, e.linkedCloud, subscriptionManager, nil, device)
		if err != nil {
			log.Errorf("cannot publish device %v resources for linkedAccount(%v): %v", e.deviceID, e.linkedAccount, err)
			h.registeredDevices.Delete(key)
		}
	}()
}

func (h *StaticDeviceEvents) Run(ctx context.Context, subscriptionManager *SubscriptionManager) {
	for {
		select {
		case e := <-h.pullDeviceChan:
			h.pullDevice(ctx, e, subscriptionManager)
		case <-ctx.Done():
			h.wg.Wait()
			return
		}
	}
}

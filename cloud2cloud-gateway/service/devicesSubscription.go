package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

type devicesSubscription struct {
	rh              *RequestHandler
	goroutinePoolGo GoroutinePoolGoFunc
}

func newDevicesSubscription(rh *RequestHandler, goroutinePoolGo GoroutinePoolGoFunc) *devicesSubscription {
	return &devicesSubscription{
		rh:              rh,
		goroutinePoolGo: goroutinePoolGo,
	}
}

func handleSubscription(ctx context.Context, rh *RequestHandler, sub store.DevicesSubscription) error {
	devicesRegistered := make(map[string]events.Device)
	devicesOnline := make(map[string]events.Device)
	devicesOffline := make(map[string]events.Device)
	devices, err := rh.GetDevices(kitNetGrpc.CtxWithToken(ctx, sub.AccessToken), nil)
	if err != nil {
		sub, errPop := rh.store.PopSubscription(ctx, sub.ID)
		if errPop != nil {
			return err
		}
		_, _ = emitEvent(ctx, events.EventType_SubscriptionCanceled, sub, func(ctx context.Context, subscriptionID string) (uint64, error) {
			return sub.SequenceNumber, nil
		}, nil)
		return err
	}
	lastDevicesRegistered := make(events.DevicesRegistered, 0, len(devices))
	lastDevicesOnline := make(events.DevicesOnline, 0, len(devices))
	lastDevicesOffline := make(events.DevicesOffline, 0, len(devices))

	for _, dev := range devices {
		devicesRegistered[dev.Device.ID] = events.Device{
			ID: dev.Device.ID,
		}
		lastDevicesRegistered = append(lastDevicesRegistered, events.Device{
			ID: dev.Device.ID,
		})
		if dev.Status == Status_ONLINE {
			devicesOnline[dev.Device.ID] = events.Device{
				ID: dev.Device.ID,
			}
			lastDevicesOnline = append(lastDevicesOnline, events.Device{
				ID: dev.Device.ID,
			})
		} else {
			devicesOffline[dev.Device.ID] = events.Device{
				ID: dev.Device.ID,
			}
			lastDevicesOffline = append(lastDevicesOffline, events.Device{
				ID: dev.Device.ID,
			})
		}
	}

	devicesUnregistered := make(map[string]events.Device)
	for _, dev := range sub.LastDevicesRegistered {
		dev, ok := devicesRegistered[dev.ID]
		if ok {
			delete(devicesRegistered, dev.ID)
		} else {
			devicesUnregistered[dev.ID] = dev
		}
	}

	for _, dev := range sub.LastDevicesOnline {
		delete(devicesOnline, dev.ID)
	}
	for _, dev := range sub.LastDevicesOffline {
		delete(devicesOffline, dev.ID)
	}

	if sub.SequenceNumber != 0 && len(devicesRegistered) == 0 && len(devicesUnregistered) == 0 && len(devicesOnline) == 0 && len(devicesOffline) == 0 {
		return nil
	}

	log.Debugf("emit events for subscription %+v", sub)

	for _, eventType := range sub.EventTypes {
		switch eventType {
		case events.EventType_DevicesRegistered:
			if len(devicesRegistered) > 0 || sub.SequenceNumber == 0 {
				devs := make(events.DevicesRegistered, 0, len(devicesRegistered))
				for _, dev := range devicesRegistered {
					devs = append(devs, dev)
				}
				remove, err := emitEvent(ctx, eventType, sub.Subscription, rh.store.IncrementSubscriptionSequenceNumber, devs)
				if remove {
					rh.store.PopSubscription(ctx, sub.ID)
				}
				if err != nil {
					return fmt.Errorf("cannot emit event: %w", err)
				}
			}
		case events.EventType_DevicesUnregistered:
			if len(devicesUnregistered) > 0 || sub.SequenceNumber == 0 {
				devs := make(events.DevicesUnregistered, 0, len(devicesUnregistered))
				for _, dev := range devicesUnregistered {
					devs = append(devs, dev)
				}
				remove, err := emitEvent(ctx, eventType, sub.Subscription, rh.store.IncrementSubscriptionSequenceNumber, devs)
				if remove {
					rh.store.PopSubscription(ctx, sub.ID)
				}
				if err != nil {
					return fmt.Errorf("cannot emit event: %w", err)
				}
			}

		case events.EventType_DevicesOnline:
			if len(devicesOnline) > 0 || sub.SequenceNumber == 0 {
				devs := make(events.DevicesOnline, 0, len(devicesOnline))
				for _, dev := range devicesOnline {
					devs = append(devs, dev)
				}
				remove, err := emitEvent(ctx, eventType, sub.Subscription, rh.store.IncrementSubscriptionSequenceNumber, devs)
				if remove {
					rh.store.PopSubscription(ctx, sub.ID)
				}
				if err != nil {
					return fmt.Errorf("cannot emit event: %w", err)
				}
			}
		case events.EventType_DevicesOffline:
			if len(devicesOffline) > 0 || sub.SequenceNumber == 0 {
				devs := make(events.DevicesOffline, 0, len(devicesOffline))
				for _, dev := range devicesOffline {
					devs = append(devs, dev)
				}
				remove, err := emitEvent(ctx, eventType, sub.Subscription, rh.store.IncrementSubscriptionSequenceNumber, devs)
				if remove {
					rh.store.PopSubscription(ctx, sub.ID)
				}
				if err != nil {
					return fmt.Errorf("cannot emit event: %w", err)
				}
			}
		}
	}

	err = rh.store.UpdateDevicesSubscription(ctx, sub.ID, lastDevicesRegistered, lastDevicesOnline, lastDevicesOffline)
	if err != nil {
		return err
	}
	return nil
}

func (s *devicesSubscription) Handle(ctx context.Context, iter store.DevicesSubscriptionIter) error {
	var wg sync.WaitGroup
	for {
		var sub store.DevicesSubscription
		if !iter.Next(ctx, &sub) {
			break
		}
		wg.Add(1)
		err := s.goroutinePoolGo(func() {
			defer wg.Done()
			err := handleSubscription(ctx, s.rh, sub)
			if err != nil {
				log.Error(fmt.Errorf("cannot handle subscription %v: %w", sub.ID, err))
			}
		})
		if err != nil {
			wg.Done()
		}
	}
	wg.Wait()
	return iter.Err()
}

func (s *devicesSubscription) Serve(ctx context.Context, checkInterval time.Duration) {
	for {
		err := s.rh.store.LoadDevicesSubscriptions(ctx, store.DevicesSubscriptionQuery{
			LastCheck: time.Now().Add(-checkInterval),
		}, s)
		if err != nil {
			log.Errorf("cannot server devicesscriptionSubscription: %v", err)
		}
		select {
		case <-time.After(checkInterval):
		case <-ctx.Done():
			return
		}
	}
}

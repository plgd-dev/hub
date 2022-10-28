package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
)

// Obtain deviceObserver from the client.
//
// The function might block and wait for the deviceObserver to be initialized.
func (c *session) getDeviceObserver(ctx context.Context) (*observation.DeviceObserver, error) {
	getError := func(err error) error {
		return fmt.Errorf("cannot get device observer: %w", err)
	}

	var deviceObserverFut *future.Future
	c.mutex.Lock()
	deviceObserverFut = c.deviceObserver
	c.mutex.Unlock()

	if deviceObserverFut == nil {
		return nil, fmt.Errorf("observer not set")
	}
	v, err := deviceObserverFut.Get(ctx)
	if err != nil {
		return nil, getError(err)
	}
	deviceObserver, ok := v.(*observation.DeviceObserver)
	if !ok {
		return nil, getError(fmt.Errorf("invalid future value"))
	}
	return deviceObserver, nil
}

// Replace deviceObserver instance in the client.
func (c *session) replaceDeviceObserver(deviceObserverFuture *future.Future) *future.Future {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	prevDeviceObserver := c.deviceObserver
	c.deviceObserver = deviceObserverFuture
	return prevDeviceObserver
}

// Replace deviceObserver instance in the client if Device Twin setting was changed for the device.
func (c *session) replaceDeviceObserverWithDeviceTwin(ctx context.Context, twinEnabled bool) (bool, error) {
	obs, err := c.getDeviceObserver(ctx)
	if err != nil {
		return false, err
	}
	prevTwinEnabled := obs.GetTwinEnabled()
	deviceID := obs.GetDeviceID()
	observationType := obs.GetObservationType()
	if prevTwinEnabled == twinEnabled {
		return prevTwinEnabled, nil
	}
	deviceObserverFuture, setDeviceObserver := future.New()
	oldDeviceObserver := c.replaceDeviceObserver(deviceObserverFuture)
	if err := cleanDeviceObserver(ctx, oldDeviceObserver); err != nil {
		c.Errorf("failed to close replaced device observer: %w", err)
	}

	deviceObserver, err := observation.NewDeviceObserver(c.Context(), deviceID, c, c, c,
		observation.MakeResourcesObserverCallbacks(c.onObserveResource, c.onGetResourceContent, c.UpdateTwinSynchronizationStatus),
		observation.WithTwinSynchronization(twinEnabled), observation.WithObservationType(observationType),
		observation.WithLogger(c.getLogger()),
		observation.WithObservationPerResourceEnabled(c.server.config.APIs.COAP.ObservationPerResourceEnabled),
	)
	if err != nil {
		setDeviceObserver(nil, err)
		return false, fmt.Errorf("cannot create observer for device %v: %w", deviceID, err)
	}
	setDeviceObserver(deviceObserver, nil)
	return prevTwinEnabled, nil
}

func toDeviceObserver(ctx context.Context, devObsFut *future.Future) (*observation.DeviceObserver, error) {
	if devObsFut == nil {
		return nil, nil
	}
	v, err := devObsFut.Get(ctx)
	if err != nil {
		return nil, err
	}
	deviceObserver, ok := v.(*observation.DeviceObserver)
	if !ok {
		return nil, fmt.Errorf("invalid future value")
	}
	return deviceObserver, nil
}

// Clean observations in the given deviceObserver instance.
func cleanDeviceObserver(ctx context.Context, devObsFut *future.Future) error {
	deviceObserver, err := toDeviceObserver(ctx, devObsFut)
	if err != nil {
		return fmt.Errorf("cannot clean device observer: %w", err)
	}
	if deviceObserver == nil {
		return nil
	}
	deviceObserver.Clean(ctx)
	return nil
}

// Replace the deviceObserver instance in the client with nil and clean up the previous deviceObserver instance.
func (c *session) closeDeviceObserver(ctx context.Context) error {
	deviceObserverFut := c.replaceDeviceObserver(nil)
	return cleanDeviceObserver(ctx, deviceObserverFut)
}

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
)

// Obtain deviceObserver from the client.
//
// The function might block and wait for the deviceObserver to be initialized.
func (c *session) getDeviceObserver(ctx context.Context) (*observation.DeviceObserver, bool, error) {
	getError := func(err error) error {
		return fmt.Errorf("cannot get device observer: %w", err)
	}

	var deviceObserverFut *future.Future
	c.private.mutex.Lock()
	deviceObserverFut = c.private.deviceObserver
	c.private.mutex.Unlock()

	if deviceObserverFut == nil {
		return nil, false, nil
	}
	v, err := deviceObserverFut.Get(ctx)
	if err != nil {
		return nil, false, getError(err)
	}
	deviceObserver, ok := v.(*observation.DeviceObserver)
	if !ok {
		return nil, false, getError(errors.New("invalid future value"))
	}
	return deviceObserver, true, nil
}

// Replace deviceObserver instance in the client.
func (c *session) replaceDeviceObserver(deviceObserverFuture *future.Future) *future.Future {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	prevDeviceObserver := c.private.deviceObserver
	c.private.deviceObserver = deviceObserverFuture
	return prevDeviceObserver
}

// Replace deviceObserver instance in the client if Device Twin setting was changed for the device.
func (c *session) replaceDeviceObserverWithDeviceTwin(ctx context.Context, twinEnabled, twinForceSynchronization bool) (bool, error) {
	obs, ok, err := c.getDeviceObserver(ctx)
	if err != nil {
		return false, err
	}
	deviceID := c.deviceID()
	prevTwinEnabled := false
	observationType := observation.ObservationType_Detect
	if ok {
		deviceID = obs.GetDeviceID()
		prevTwinEnabled = obs.GetTwinEnabled()
		observationType = obs.GetObservationType()
	}
	if deviceID == "" {
		return false, errors.New("cannot replace device observer: invalid device id")
	}
	twinEnabled = twinEnabled || twinForceSynchronization
	if !twinForceSynchronization && prevTwinEnabled == twinEnabled {
		return prevTwinEnabled, nil
	}
	deviceObserverFuture, setDeviceObserver := future.New()
	oldDeviceObserver := c.replaceDeviceObserver(deviceObserverFuture)
	if errD := cleanDeviceObserver(ctx, oldDeviceObserver); errD != nil {
		c.Errorf("failed to close replaced device observer: %w", errD)
	}

	deviceObserver, err := observation.NewDeviceObserver(c.Context(), deviceID, c, c, c,
		observation.MakeResourcesObserverCallbacks(c.onObserveResource, c.onGetResourceContent, c.UpdateTwinSynchronizationStatus),
		observation.WithTwinEnabled(twinEnabled), observation.WithObservationType(observationType),
		observation.WithLogger(c.getLogger()),
		observation.WithRequireBatchObserveEnabled(c.server.config.APIs.COAP.RequireBatchObserveEnabled),
		observation.WithMaxETagsCountInRequest(c.server.config.DeviceTwin.MaxETagsCountInRequest),
		observation.WithUseETags(!twinForceSynchronization && c.server.config.DeviceTwin.UseETags),
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
		return nil, errors.New("invalid future value")
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

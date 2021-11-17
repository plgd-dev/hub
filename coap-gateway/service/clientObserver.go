package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/coap-gateway/service/observation"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/sync/task/future"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
)

func (client *Client) getDeviceObserver(ctx context.Context) (*observation.DeviceObserver, error) {
	getError := func(err error) error {
		return fmt.Errorf("cannot get device observer: %w", err)
	}

	var deviceObserverFut *future.Future
	client.mutex.Lock()
	deviceObserverFut = client.deviceObserver
	client.mutex.Unlock()

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

func (client *Client) replaceDeviceObserver(deviceObserver *observation.DeviceObserver) *future.Future {
	var newDeviceObserverFut *future.Future
	if deviceObserver != nil {
		var setDeviceObserver future.SetFunc
		newDeviceObserverFut, setDeviceObserver = future.New()
		setDeviceObserver(deviceObserver, nil)
	}
	client.mutex.Lock()
	defer client.mutex.Unlock()
	prevDeviceObserver := client.deviceObserver
	client.deviceObserver = newDeviceObserverFut
	return prevDeviceObserver
}

func (client *Client) replaceDeviceObserverWithDeviceShadow(ctx context.Context, shadow commands.ShadowSynchronization) (commands.ShadowSynchronization, error) {
	obs, err := client.getDeviceObserver(ctx)
	if err != nil {
		return commands.ShadowSynchronization_UNSET, err
	}
	prevShadow := obs.GetShadowSynchronization()
	deviceID := obs.GetDeviceID()
	if prevShadow == shadow {
		return prevShadow, nil
	}
	deviceObserver, err := observation.NewDeviceObserver(ctx, deviceID, client.coapConn, client.server.rdClient,
		client.onObserveResource, client.onGetResourceContent)
	if err != nil {
		return commands.ShadowSynchronization_UNSET, fmt.Errorf("cannot create observer for device %v: %w", deviceID, err)
	}
	oldDeviceObserver := client.replaceDeviceObserver(deviceObserver)
	if err := cleanDeviceObserver(ctx, oldDeviceObserver); err != nil {
		log.Errorf("failed to close replaced device observer: %w", err)
	}
	return prevShadow, nil
}

func cleanDeviceObserver(ctx context.Context, devObsFut *future.Future) error {
	if devObsFut == nil {
		return nil
	}
	v, err := devObsFut.Get(ctx)
	if err != nil {
		return fmt.Errorf("cannot close device observer: %w", err)
	}
	deviceObserver, ok := v.(*observation.DeviceObserver)
	if !ok {
		return fmt.Errorf("cannot close device observer: invalid future value")
	}
	deviceObserver.Clean(ctx)
	return nil
}

func (client *Client) closeDeviceObserver(ctx context.Context) error {
	deviceObserverFut := client.replaceDeviceObserver(nil)
	return cleanDeviceObserver(ctx, deviceObserverFut)
}

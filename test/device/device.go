/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package device

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	deviceCoap "github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/hub/v2/test/sdk"
)

type Type int

const (
	OCF Type = iota
	Bridged
)

type Device interface {
	// GetType returns device type
	GetType() Type

	// GetID returns device ID
	GetID() string

	// SetID sets device ID
	SetID(id string)

	// GetName returns device name
	GetName() string

	// GetRetryInterval returns retry interval of the device before retrying provisioning
	GetRetryInterval(attempt int) time.Duration

	// GetDefaultResources returns default device resources
	GetDefaultResources() schema.ResourceLinks

	// GetSDKClientOptions returns options for the SDK client used with this device
	GetSDKClientOptions() []sdk.Option
}

type BaseDevice struct {
	id   string
	name string
}

func MakeBaseDevice(id, name string) BaseDevice {
	return BaseDevice{
		id:   id,
		name: name,
	}
}

func (bd *BaseDevice) GetID() string {
	return bd.id
}

func (bd *BaseDevice) SetID(id string) {
	bd.id = id
}

func (bd *BaseDevice) GetName() string {
	return bd.name
}

func (bd *BaseDevice) GetSDKClientOptions() []sdk.Option {
	return nil
}

type GetResourceOpts func(*core.Device) deviceCoap.OptionFunc

func FindDeviceByName(ctx context.Context, name string, getResourceOpts ...GetResourceOpts) (deviceID string, _ error) {
	client := core.NewClient()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := findDeviceIDByNameHandler{
		name:               name,
		cancel:             cancel,
		getResourceOptions: getResourceOpts,
	}

	err := client.GetDevicesByMulticast(ctx, core.DefaultDiscoveryConfiguration(), &h)
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	id, ok := h.id.Load().(string)
	if !ok || id == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return id, nil
}

type findDeviceIDByNameHandler struct {
	id                 atomic.Value
	name               string
	cancel             context.CancelFunc
	getResourceOptions []GetResourceOpts
}

func (h *findDeviceIDByNameHandler) Handle(ctx context.Context, dev *core.Device) {
	defer func() {
		if errC := dev.Close(ctx); errC != nil {
			h.Error(errC)
		}
	}()
	deviceLinks, err := dev.GetResourceLinks(ctx, dev.GetEndpoints())
	if err != nil {
		h.Error(err)
		return
	}
	l, ok := deviceLinks.GetResourceLink(device.ResourceURI)
	if !ok {
		return
	}
	var d device.Device
	var getResourceOpts []deviceCoap.OptionFunc
	if h.getResourceOptions != nil {
		for _, opts := range h.getResourceOptions {
			getResourceOpts = append(getResourceOpts, opts(dev))
		}
	}
	err = dev.GetResource(ctx, l, &d, getResourceOpts...)
	if err != nil {
		h.Error(err)
		return
	}
	if d.Name == h.name {
		h.id.Store(d.ID)
		h.cancel()
	}
}

func (h *findDeviceIDByNameHandler) Error(err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	log.Printf("find device ID by name handler error: %v", err.Error())
}

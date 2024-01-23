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

package bridge

import (
	"time"

	"github.com/plgd-dev/device/v2/schema"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/hub/v2/test/device"
	"github.com/plgd-dev/hub/v2/test/sdk"
)

var TestResources = []schema.ResourceLink{
	{
		Href:          schemaDevice.ResourceURI,
		ResourceTypes: []string{"oic.d.virtual", schemaDevice.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
		Policy: &schema.Policy{
			BitMask: 1,
		},
	},
	{
		Href:          maintenance.ResourceURI,
		ResourceTypes: []string{maintenance.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW},
		Policy: &schema.Policy{
			BitMask: 1,
		},
	},
}

type Device struct {
	device.BaseDevice
}

func NewDevice(id, name string) *Device {
	return &Device{
		BaseDevice: device.MakeBaseDevice(id, name),
	}
}

func (d *Device) GetSDKClientOptions() []sdk.Option {
	return []sdk.Option{sdk.WithUseDeviceIDInQuery(true)}
}

func (d *Device) GetRetryInterval(int) time.Duration {
	return time.Second * 10
}

func (d *Device) GetDefaultResources() schema.ResourceLinks {
	return TestResources
}

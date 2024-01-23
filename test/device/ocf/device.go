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

package ocf

import (
	"math"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/test/device"
)

var TestResources = []schema.ResourceLink{
	{
		Href:          platform.ResourceURI,
		ResourceTypes: []string{platform.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          schemaDevice.ResourceURI,
		ResourceTypes: []string{types.DEVICE_CLOUD, schemaDevice.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          configuration.ResourceURI,
		ResourceTypes: []string{configuration.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          "/light/1",
		ResourceTypes: []string{types.CORE_LIGHT},
		Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          "/switches",
		ResourceTypes: []string{collection.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          maintenance.ResourceURI,
		ResourceTypes: []string{maintenance.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 1,
		},
	},

	{
		Href:          plgdtime.ResourceURI,
		ResourceTypes: []string{plgdtime.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
		},
	},

	{
		Href:          softwareupdate.ResourceURI,
		ResourceTypes: []string{softwareupdate.ResourceType},
		Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: 3,
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

func (d *Device) GetRetryInterval(attempt int) time.Duration {
	/* [2s, 4s, 8s, 16s, 32s, 64s] */
	return time.Duration(math.Exp2(float64(attempt))) * time.Second
}

func (d *Device) GetDefaultResources() schema.ResourceLinks {
	return TestResources
}

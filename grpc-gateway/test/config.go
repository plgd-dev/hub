package test

import (
	"fmt"

	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

var (
	TestDeviceName string

	TestDevsimResources        []schema.ResourceLink
	TestDevsimBackendResources []schema.ResourceLink
)

func init() {
	TestDeviceName = "devsim-" + MustGetHostname()
	TestDevsimResources = []schema.ResourceLink{
		{
			Href:          "/oic/p",
			ResourceTypes: []string{"oic.wk.p"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
			Policy: schema.Policy{
				BitMask: 1,
			},
		},

		{
			Href:          "/oic/d",
			ResourceTypes: []string{"oic.d.cloudDevice", "oic.wk.d"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
			Policy: schema.Policy{
				BitMask: 1,
			},
		},

		{
			Href:          "/oc/con",
			ResourceTypes: []string{"oic.wk.con"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
			Policy: schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          "/light/1",
			ResourceTypes: []string{"core.light"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
			Policy: schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          "/light/2",
			ResourceTypes: []string{"core.light"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
			Policy: schema.Policy{
				BitMask: 3,
			},
		},
	}

	TestDevsimBackendResources = []schema.ResourceLink{
		{
			Href:          cloud.StatusHref,
			ResourceTypes: cloud.StatusResourceTypes,
			Interfaces:    cloud.StatusInterfaces,
			Policy: schema.Policy{
				BitMask: 3,
			},
			Title: "Cloud device status",
		},
	}
}

func FindResourceLink(href string) schema.ResourceLink {
	for _, l := range TestDevsimResources {
		if l.Href == href {
			return l
		}
	}
	for _, l := range TestDevsimBackendResources {
		if l.Href == href {
			return l
		}
	}
	panic(fmt.Sprintf("resource %v: not found", href))
}

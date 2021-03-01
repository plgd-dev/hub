package service

import (
	"context"
	"fmt"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	projectionRA "github.com/plgd-dev/cloud/resource-aggregate/cqrs/projection"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/kit/strings"
)

// hasMatchingType returns true for matching a resource type.
// An empty typeFilter matches all resource types.
func hasMatchingType(resourceTypes []string, typeFilter strings.Set) bool {
	if len(typeFilter) == 0 {
		return true
	}
	if len(resourceTypes) == 0 {
		return false
	}
	return typeFilter.HasOneOf(resourceTypes...)
}

type Projection struct {
	*projectionRA.Projection
	cache *cache.Cache
}

type Resource struct {
	Projection *resourceProjection
	Resource   *commands.Resource
}

func NewProjection(ctx context.Context, name string, store eventstore.EventStore, subscriber eventbus.Subscriber, newModelFunc eventstore.FactoryModelFunc, expiration time.Duration) (*Projection, error) {
	projection, err := projectionRA.NewProjection(ctx, name, store, subscriber, newModelFunc)
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	cache := cache.New(expiration, expiration)
	cache.OnEvicted(func(deviceID string, _ interface{}) {
		projection.Unregister(deviceID)
	})
	return &Projection{Projection: projection, cache: cache}, nil
}

func (p *Projection) getModels(ctx context.Context, resourceID *commands.ResourceId) ([]eventstore.Model, error) {
	loaded, err := p.Register(ctx, resourceID.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("cannot register to projection for %v: %w", resourceID, err)
	}
	if loaded {
		p.cache.Set(resourceID.GetDeviceId(), loaded, cache.DefaultExpiration)
	} else {
		defer func(ID string) {
			p.Unregister(ID)
		}(resourceID.GetDeviceId())
	}
	m := p.Models(resourceID)
	if !loaded && len(m) == 0 {
		err := p.ForceUpdate(ctx, resourceID)
		if err == nil {
			m = p.Models(resourceID)
		}
	}
	return m, nil
}

func (p *Projection) GetResourceLinks(ctx context.Context, deviceIDFilter, typeFilter strings.Set) (map[string]map[string]*commands.Resource, error) {
	devicesResourceLinks := make(map[string]map[string]*commands.Resource)
	for deviceID := range deviceIDFilter {
		models, err := p.getModels(ctx, commands.NewResourceID(deviceID, commands.ResourceLinksHref))
		if err != nil {
			return nil, err
		}
		if len(models) != 1 {
			return nil, nil
		}
		resourceLinks := models[0].(*resourceLinksProjection).Clone()
		resources := make(map[string]*commands.Resource)
		for key, resource := range resourceLinks.resources {
			if !hasMatchingType(resource.ResourceTypes, typeFilter) {
				continue
			}
			resources[key] = resource
		}
		devicesResourceLinks[resourceLinks.deviceID] = resources
	}

	return devicesResourceLinks, nil
}

func normalizeResourceIDs(resourceIDFilter []*commands.ResourceId) []*commands.ResourceId {
	if len(resourceIDFilter) == 0 {
		return nil
	}
	resourceIDs := make(map[string]map[string]bool)
	for _, resourceID := range resourceIDFilter {
		v, ok := resourceIDs[resourceID.GetDeviceId()]
		if !ok {
			v = make(map[string]bool)
			resourceIDs[resourceID.GetDeviceId()] = v
			if resourceID.GetHref() != "" {
				v[resourceID.GetHref()] = true
			}
			continue
		}
		if len(v) == 0 {
			continue
		}
		if resourceID.GetHref() == "" {
			resourceIDs[resourceID.GetDeviceId()] = make(map[string]bool)
		} else {
			v[resourceID.GetHref()] = true
		}
	}
	resourceIDFilter = make([]*commands.ResourceId, 0, len(resourceIDFilter))
	for deviceID, hrefs := range resourceIDs {
		if len(hrefs) == 0 {
			resourceIDFilter = append(resourceIDFilter, &commands.ResourceId{DeviceId: deviceID})
			continue
		}
		for href := range hrefs {
			resourceIDFilter = append(resourceIDFilter, &commands.ResourceId{DeviceId: deviceID, Href: href})
		}
	}
	return resourceIDFilter
}

func (p *Projection) GetResourcesDetails(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set) (map[string]map[string]*Resource, error) {
	unusedResourceLinks, resources, err := p.getResources(ctx, resourceIDFilter, typeFilter)
	if err != nil {
		return nil, err
	}

	for deviceID, links := range unusedResourceLinks {
		for href, link := range links {
			deviceHrefs, ok := resources[deviceID]
			if !ok {
				deviceHrefs = make(map[string]*Resource)
				resources[deviceID] = deviceHrefs
			}
			deviceHrefs[href] = &Resource{Resource: link}
		}
	}

	return resources, nil
}

func (p *Projection) GetResources(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set) (map[string]map[string]*Resource, error) {
	_, resources, err := p.getResources(ctx, resourceIDFilter, typeFilter)
	return resources, err
}

func (p *Projection) getResources(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set) (map[string]map[string]*commands.Resource, map[string]map[string]*Resource, error) {
	resourceIDFilter = normalizeResourceIDs(resourceIDFilter)
	resourceLinks := make(map[string]map[string]*commands.Resource)

	models := make([]eventstore.Model, 0, 32)
	for _, rid := range resourceIDFilter {
		// build resource links map of all devices which are requested
		if _, present := resourceLinks[rid.GetDeviceId()]; !present {
			rl, err := p.GetResourceLinks(ctx, strings.Set{rid.GetDeviceId(): {}}, nil)
			if err != nil {
				return nil, nil, err
			}
			resourceLinks[rid.GetDeviceId()] = rl[rid.GetDeviceId()]
		}
		m, err := p.getModels(ctx, rid)
		if err != nil {
			return nil, nil, err
		}
		models = append(models, m...)
	}

	unusedResourceLinks := make(map[string]map[string]*commands.Resource)
	for _, rid := range resourceIDFilter {
		links, ok := unusedResourceLinks[rid.GetDeviceId()]
		if !ok {
			links = make(map[string]*commands.Resource)
			unusedResourceLinks[rid.GetDeviceId()] = links
		}
		if rid.GetHref() == "" {
			for href, link := range resourceLinks[rid.GetDeviceId()] {
				if !hasMatchingType(link.ResourceTypes, typeFilter) {
					continue
				}
				links[href] = link
			}
			continue
		}
		link, ok := resourceLinks[rid.GetDeviceId()][rid.GetHref()]
		if ok {
			if !hasMatchingType(link.ResourceTypes, typeFilter) {
				continue
			}
			links[rid.GetHref()] = link
		}
	}

	resources := make(map[string]map[string]*Resource)
	for _, m := range models {
		if m.SnapshotEventType() == events.NewResourceLinksSnapshotTaken().SnapshotEventType() {
			continue
		}
		rp := m.(*resourceProjection).Clone()
		if _, present := resourceLinks[rp.resourceId.GetDeviceId()][rp.resourceId.GetHref()]; !present {
			continue
		}

		if !hasMatchingType(resourceLinks[rp.resourceId.GetDeviceId()][rp.resourceId.GetHref()].ResourceTypes, typeFilter) {
			continue
		}

		deviceHrefs, ok := resources[rp.resourceId.GetDeviceId()]
		if !ok {
			deviceHrefs = make(map[string]*Resource)
			resources[rp.resourceId.GetDeviceId()] = deviceHrefs
		}
		deviceHrefs[rp.resourceId.GetHref()] = &Resource{Projection: rp, Resource: resourceLinks[rp.resourceId.GetDeviceId()][rp.resourceId.GetHref()]}
		delete(unusedResourceLinks[rp.resourceId.GetDeviceId()], rp.resourceId.GetHref())
	}

	return unusedResourceLinks, resources, nil
}

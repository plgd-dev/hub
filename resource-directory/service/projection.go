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

func (p *Projection) GetResources(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set) (map[string]map[string]*Resource, error) {
	resourceLinks := make(map[string]map[string]*commands.Resource)
	models := make([]eventstore.Model, 0, 32)
	for _, rid := range resourceIDFilter {
		// build resource links map of all devices which are requested
		if _, present := resourceLinks[rid.GetDeviceId()]; !present {
			rl, err := p.GetResourceLinks(ctx, strings.Set{rid.GetDeviceId(): {}}, nil)
			if err != nil {
				return nil, err
			}
			resourceLinks[rid.GetDeviceId()] = rl[rid.GetDeviceId()]
		}

		m, err := p.getModels(ctx, rid)
		if err != nil {
			return nil, err
		}
		models = append(models, m...)
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
	}

	return resources, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	cache "github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
	projectionRA "github.com/plgd-dev/hub/resource-aggregate/cqrs/projection"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/kit/v2/strings"
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
	expiration time.Duration
	cache      *cache.Cache
}

func NewProjection(ctx context.Context, name string, store eventstore.EventStore, subscriber eventbus.Subscriber, newModelFunc eventstore.FactoryModelFunc, expiration time.Duration) (*Projection, error) {
	projection, err := projectionRA.NewProjection(ctx, name, store, subscriber, newModelFunc)
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	cleanupInterval := expiration / 30
	if cleanupInterval < time.Minute {
		cleanupInterval = expiration
		if cleanupInterval > time.Minute {
			cleanupInterval = time.Minute
		}
	}
	cache := cache.NewCache()
	add := periodic.New(ctx.Done(), cleanupInterval)
	add(func(now time.Time) bool {
		cache.CheckExpirations(now)
		return true
	})
	return &Projection{Projection: projection, cache: cache, expiration: expiration}, nil
}

func (p *Projection) getModels(ctx context.Context, resourceID *commands.ResourceId, expectedModels int) ([]eventstore.Model, error) {
	created, err := p.Register(ctx, resourceID.GetDeviceId())
	if err != nil {
		return nil, fmt.Errorf("cannot register to projection for %v: %w", resourceID, err)
	}
	p.cache.Delete(resourceID.GetDeviceId())
	p.cache.LoadOrStore(resourceID.GetDeviceId(), cache.NewElement(resourceID.GetDeviceId(), time.Now().Add(p.expiration), func(d interface{}) {
		deviceID := d.(string)
		if err := p.Unregister(deviceID); err != nil {
			log.Errorf("failed to unregister device %v in projection cache during eviction: %w", deviceID, err)
		}
	}))
	if !created {
		defer func(ID string) {
			if err := p.Unregister(ID); err != nil {
				log.Errorf("failed to unregister device %v in projection cache after replacement: %w", ID, err)
			}
		}(resourceID.GetDeviceId())
	}
	m := p.Models(resourceID)
	if !created && len(m) < expectedModels {
		err := p.ForceUpdate(ctx, resourceID)
		if err == nil {
			m = p.Models(resourceID)
		}
	}
	return m, nil
}

func (p *Projection) GetResourceLinks(ctx context.Context, deviceIDFilter, typeFilter strings.Set) (map[string]*events.ResourceLinksSnapshotTaken, error) {
	devicesResourceLinks := make(map[string]*events.ResourceLinksSnapshotTaken)
	for deviceID := range deviceIDFilter {
		models, err := p.getModels(ctx, commands.NewResourceID(deviceID, commands.ResourceLinksHref), 1)
		if err != nil {
			return nil, err
		}
		if len(models) != 1 {
			continue
		}
		resourceLinks := models[0].(*resourceLinksProjection).Clone()
		devicesResourceLinks[resourceLinks.snapshot.GetDeviceId()] = resourceLinks.snapshot
		for href, resource := range resourceLinks.snapshot.GetResources() {
			if !hasMatchingType(resource.ResourceTypes, typeFilter) {
				delete(resourceLinks.snapshot.Resources, href)
			}
		}
	}

	return devicesResourceLinks, nil
}

func (p *Projection) GetDevicesMetadata(ctx context.Context, deviceIDFilter strings.Set) (map[string]*events.DeviceMetadataSnapshotTaken, error) {
	devicesMetadata := make(map[string]*events.DeviceMetadataSnapshotTaken)
	for deviceID := range deviceIDFilter {
		models, err := p.getModels(ctx, commands.NewResourceID(deviceID, commands.StatusHref), 1)
		if err != nil {
			return nil, err
		}
		if len(models) != 1 {
			continue
		}
		deviceMetadata := models[0].(*deviceMetadataProjection).Clone()
		deviceID = deviceMetadata.data.GetDeviceId()
		devicesMetadata[deviceID] = deviceMetadata.data
	}

	return devicesMetadata, nil
}

// Group filter first by device ID and then by resource ID
func getResourceIDMapFilter(resourceIDFilter []*commands.ResourceId) map[string]map[string]bool {
	resourceIDMapFilter := make(map[string]map[string]bool)
	for _, resourceID := range resourceIDFilter {
		if resourceID.GetHref() == "" {
			resourceIDMapFilter[resourceID.GetDeviceId()] = nil
			continue
		}
		hrefs, present := resourceIDMapFilter[resourceID.GetDeviceId()]
		if present && hrefs == nil {
			continue
		}
		if !present {
			resourceIDMapFilter[resourceID.GetDeviceId()] = make(map[string]bool)
		}
		resourceIDMapFilter[resourceID.GetDeviceId()][resourceID.GetHref()] = true
	}
	return resourceIDMapFilter
}

func (p *Projection) GetResourcesWithLinks(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set) (map[string]map[string]*Resource, error) {
	resourceIDMapFilter := getResourceIDMapFilter(resourceIDFilter)
	resources := make(map[string]map[string]*Resource)
	models := make([]eventstore.Model, 0, len(resourceIDMapFilter))
	for deviceID, hrefs := range resourceIDMapFilter {
		// build resource links map of all devices which are requested
		rl, err := p.GetResourceLinks(ctx, strings.Set{deviceID: {}}, nil)
		if err != nil {
			return nil, err
		}

		anyDeviceResourceFound := false
		expectedModels := len(rl[deviceID].GetResources()) + 2 // for metadata and resourcelinks
		resources[deviceID] = make(map[string]*Resource)
		if hrefs == nil {
			// case when client requests all device resources
			for _, resource := range rl[deviceID].GetResources() {
				if hasMatchingType(resource.ResourceTypes, typeFilter) {
					resources[deviceID][resource.GetHref()] = &Resource{Resource: resource}
					anyDeviceResourceFound = true
				}
			}
		} else {
			// case when client requests specific device resource
			for href := range hrefs {
				if resource, present := rl[deviceID].GetResources()[href]; present {
					if hasMatchingType(resource.ResourceTypes, typeFilter) {
						resources[deviceID][href] = &Resource{Resource: resource}
						anyDeviceResourceFound = true
					}
				}
			}
		}

		if anyDeviceResourceFound {
			m, err := p.getModels(ctx, commands.NewResourceID(deviceID, ""), expectedModels)
			if err != nil {
				return nil, err
			}
			models = append(models, m...)
		} else {
			delete(resources, deviceID)
		}
	}

	for _, m := range models {
		if m.(interface{ EventType() string }).EventType() == events.NewResourceLinksSnapshotTaken().EventType() {
			continue
		}
		rp := m.(*resourceProjection).Clone()
		if _, present := resources[rp.resourceID.GetDeviceId()][rp.resourceID.GetHref()]; !present {
			continue
		}
		resources[rp.resourceID.GetDeviceId()][rp.resourceID.GetHref()].projection = rp
	}

	return resources, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	cache "github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	projectionRA "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/projection"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
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
	cache      *cache.Cache[string, string]
}

func NewProjection(ctx context.Context, name string, store eventstore.EventStore, subscriber eventbus.Subscriber, newModelFunc eventstore.FactoryModelFunc, expiration time.Duration) (*Projection, error) {
	projection, err := projectionRA.NewProjection(ctx, name, store, subscriber, newModelFunc)
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	cleanupInterval := expiration / 2
	if cleanupInterval < time.Second {
		cleanupInterval = expiration
	}
	if cleanupInterval > time.Minute {
		cleanupInterval = time.Minute
	}
	cache := cache.NewCache[string, string]()
	add := periodic.New(ctx.Done(), cleanupInterval)
	add(func(now time.Time) bool {
		cache.CheckExpirations(now)
		return true
	})
	return &Projection{Projection: projection, cache: cache, expiration: expiration}, nil
}

func (p *Projection) LoadResourceLinks(deviceIDFilter, toReloadDevices strings.Set, onResourceLinkProjection func(m *resourceLinksProjection) error) error {
	for deviceID := range deviceIDFilter {
		reload := true
		var err error
		p.Models(func(m eventstore.Model) (wantNext bool) {
			rl := m.(*resourceLinksProjection)
			if rl.LenResources() == 0 {
				return false
			}
			reload = false
			err = onResourceLinkProjection(rl)
			return err == nil
		}, commands.NewResourceID(deviceID, commands.ResourceLinksHref))
		if reload && toReloadDevices != nil {
			toReloadDevices.Add(deviceID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Projection) ReloadDevices(ctx context.Context, deviceIDFilter strings.Set) {
	for deviceID := range deviceIDFilter {
		created, err := p.Register(ctx, deviceID)
		if err != nil {
			log.Errorf("cannot register to projection for %v: %w", deviceID, err)
			continue
		}
		p.cache.Delete(deviceID)
		p.cache.LoadOrStore(deviceID, cache.NewElement(deviceID, time.Now().Add(p.expiration), func(deviceID string) {
			if err := p.Unregister(deviceID); err != nil {
				log.Errorf("failed to unregister device %v in projection cache during eviction: %w", deviceID, err)
			}
		}))
		if !created {
			err := p.ForceUpdate(ctx, commands.NewResourceID(deviceID, ""))
			if err != nil {
				log.Errorf("cannot update projection for device %v: %w", deviceID, err)
			}
			defer func(ID string) {
				if err := p.Unregister(ID); err != nil {
					log.Errorf("failed to unregister device %v in projection cache after replacement: %w", ID, err)
				}
			}(deviceID)
		}
	}
}

func (p *Projection) LoadDevicesMetadata(deviceIDFilter, toReloadDevices strings.Set, onDeviceMetadataProjection func(m *deviceMetadataProjection) error) error {
	var err error
	for deviceID := range deviceIDFilter {
		reload := true
		p.Models(func(m eventstore.Model) (wantNext bool) {
			dm := m.(*deviceMetadataProjection)
			if !dm.IsInitialized() {
				return true
			}
			reload = false
			err = onDeviceMetadataProjection(dm)
			return err == nil
		}, commands.NewResourceID(deviceID, commands.StatusHref))
		if reload && toReloadDevices != nil {
			toReloadDevices.Add(deviceID)
		}
	}
	return err
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

func (p *Projection) wantToReloadDevice(rl *resourceLinksProjection, hrefFilter map[string]bool, typeFilter strings.Set) bool {
	var finalReload bool
	rl.IterateOverResources(func(res *commands.Resource) (wantNext bool) {
		if len(hrefFilter) > 0 && !hrefFilter[res.GetHref()] {
			return true
		}
		if !hasMatchingType(res.GetResourceTypes(), typeFilter) {
			return true
		}
		reload := true
		p.Models(func(eventstore.Model) (wantNext bool) {
			reload = false
			return true
		}, commands.NewResourceID(rl.GetDeviceID(), res.GetHref()))
		if reload {
			finalReload = true
			return false
		}
		return true
	})
	return finalReload
}

type resourceFilter struct {
	hrefFilter map[string]bool
	typeFilter strings.Set
}

func (rf *resourceFilter) isMatchingResource(href string, resourceTypes []string) bool {
	if len(rf.hrefFilter) > 0 && !rf.hrefFilter[href] {
		return false
	}
	if !hasMatchingType(resourceTypes, rf.typeFilter) {
		return false
	}
	return true
}

func (p *Projection) loadAllDeviceResources(ctx context.Context, deviceID string, hrefFilter map[string]bool, typeFilter strings.Set, onResource func(*Resource) error) error {
	rf := resourceFilter{hrefFilter: hrefFilter, typeFilter: typeFilter}
	err := p.ForceUpdate(ctx, commands.NewResourceID(deviceID, ""))
	if err != nil {
		return err
	}
	p.GroupsModels(func(model eventstore.Model) (wantNext bool) {
		rp, ok := model.(*resourceProjection)
		if !ok {
			return true
		}
		resourceID, resourceTypes := rp.GetMatchingData()
		if !rf.isMatchingResource(resourceID.GetHref(), resourceTypes) {
			return true
		}
		err := onResource(&Resource{
			projection: rp,
			Resource: &commands.Resource{
				Href:          resourceID.GetHref(),
				ResourceTypes: resourceTypes,
				Anchor:        "ocf://" + resourceID.GetDeviceId(),
				DeviceId:      resourceID.GetDeviceId(),
			},
		})
		return err == nil
	}, deviceID)
	return nil
}

func (p *Projection) loadResourceWithLinks(deviceID string, hrefFilter map[string]bool, typeFilter strings.Set, toReloadDevices strings.Set, onResource func(*Resource) error) error {
	rf := resourceFilter{hrefFilter: hrefFilter, typeFilter: typeFilter}
	isSnapShotEvent := func(model eventstore.Model) bool {
		e, ok := model.(interface{ EventType() string })
		if !ok {
			panic(fmt.Errorf("invalid event type(%T)", model))
		}
		t := e.EventType()
		return t == events.NewResourceLinksSnapshotTaken().EventType() ||
			t == events.NewDeviceMetadataSnapshotTaken().EventType()
	}

	iterateResources := func(rl *resourceLinksProjection) error {
		var err error
		rl.IterateOverResources(func(res *commands.Resource) (wantNext bool) {
			if !rf.isMatchingResource(res.GetHref(), res.GetResourceTypes()) {
				return true
			}
			p.Models(func(model eventstore.Model) (wantNext bool) {
				if isSnapShotEvent(model) {
					return true
				}
				rp := model.(*resourceProjection)
				err = onResource(&Resource{
					projection: rp,
					Resource:   res,
				})
				return err == nil
			}, commands.NewResourceID(rl.GetDeviceID(), res.GetHref()))
			return true
		})
		return err
	}

	return p.LoadResourceLinks(strings.Set{deviceID: struct{}{}}, toReloadDevices, func(rl *resourceLinksProjection) error {
		if p.wantToReloadDevice(rl, hrefFilter, typeFilter) && toReloadDevices != nil {
			// if toReloadDevices == nil it means that Reload was executed but all resources are not available yet, we want to provide partial resoures then.
			toReloadDevices.Add(rl.GetDeviceID())
			return nil
		}
		return iterateResources(rl)
	})
}

func (p *Projection) LoadResources(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set, includeHiddenResources bool, toReloadDevices strings.Set, onResource func(*Resource) error) error {
	resourceIDMapFilter := getResourceIDMapFilter(resourceIDFilter)
	for deviceID, hrefFilter := range resourceIDMapFilter { // filter duplicit load
		if includeHiddenResources {
			err := p.loadAllDeviceResources(ctx, deviceID, hrefFilter, typeFilter, onResource)
			if err != nil {
				return err
			}
			continue
		}
		err := p.loadResourceWithLinks(deviceID, hrefFilter, typeFilter, toReloadDevices, onResource)
		if err != nil {
			return err
		}
	}
	return nil
}

package service

import (
	"context"
	"fmt"
	"time"

	cache "github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
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

func (p *Projection) LoadResourceLinks(ctx context.Context, deviceIDFilter, toReloadDevices strings.Set, onResourceLinkProjection func(m *resourceLinksProjection) error) error {
	for deviceID := range deviceIDFilter {
		reload := true
		var err error
		p.Models(func(m eventstore.Model) (wantNext bool) {
			rl := m.(*resourceLinksProjection)
			if len(rl.snapshot.GetResources()) == 0 {
				return
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
		}
		p.cache.Delete(deviceID)
		p.cache.LoadOrStore(deviceID, cache.NewElement(deviceID, time.Now().Add(p.expiration), func(d interface{}) {
			deviceID := d.(string)
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

func (p *Projection) LoadDevicesMetadata(ctx context.Context, deviceIDFilter, toReloadDevices strings.Set, onDeviceMetadataProjection func(m *deviceMetadataProjection) error) error {
	var err error
	for deviceID := range deviceIDFilter {
		reload := true
		p.Models(func(m eventstore.Model) (wantNext bool) {
			dm := m.(*deviceMetadataProjection)
			if dm.data == nil {
				return
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
	for _, res := range rl.snapshot.GetResources() {
		if len(hrefFilter) > 0 && !hrefFilter[res.GetHref()] {
			continue
		}
		if !hasMatchingType(res.ResourceTypes, typeFilter) {
			continue
		}
		reload := true
		p.Models(func(eventstore.Model) (wantNext bool) {
			reload = false
			return true
		}, commands.NewResourceID(rl.GetDeviceID(), res.Href))
		if reload {
			return true
		}
	}
	return false
}

func (p *Projection) LoadResourcesWithLinks(ctx context.Context, resourceIDFilter []*commands.ResourceId, typeFilter strings.Set, toReloadDevices strings.Set, onResource func(*Resource) error) error {
	resourceIDMapFilter := getResourceIDMapFilter(resourceIDFilter)
	for deviceID, hrefFilter := range resourceIDMapFilter { // filter duplicit load
		err := p.LoadResourceLinks(ctx, strings.Set{deviceID: struct{}{}}, toReloadDevices, func(rl *resourceLinksProjection) error {
			if p.wantToReloadDevice(rl, hrefFilter, typeFilter) {
				if toReloadDevices != nil {
					toReloadDevices.Add(rl.GetDeviceID())
				}
				return nil
			}
			for _, res := range rl.snapshot.GetResources() {
				if len(hrefFilter) > 0 && !hrefFilter[res.GetHref()] {
					continue
				}
				if !hasMatchingType(res.ResourceTypes, typeFilter) {
					continue
				}
				var err error
				p.Models(func(model eventstore.Model) (wantNext bool) {
					t := model.(interface{ EventType() string }).EventType()
					if t == events.NewResourceLinksSnapshotTaken().EventType() ||
						t == events.NewDeviceMetadataSnapshotTaken().EventType() {
						return true
					}
					rp := model.(*resourceProjection)
					err = onResource(&Resource{
						projection: rp,
						Resource:   res,
					})
					return err == nil
				}, commands.NewResourceID(rl.GetDeviceID(), res.Href))
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

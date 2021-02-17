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

func (p *Projection) GetResourceCtxs(ctx context.Context, resourceIDsFilter []*commands.ResourceId, typeFilter, deviceIDs strings.Set) (map[string]map[string]*resourceCtx, error) {
	models := make([]eventstore.Model, 0, 32)
	for _, rid := range resourceIDsFilter {
		m, err := p.getModels(ctx, rid)
		if err != nil {
			return nil, err
		}
		models = append(models, m...)
	}

	for deviceID := range deviceIDs {
		m, err := p.getModels(ctx, &commands.ResourceId{DeviceId: deviceID})
		if err != nil {
			return nil, err
		}
		models = append(models, m...)
	}

	clonedModels := make(map[string]map[string]*resourceCtx)
	for _, m := range models {
		model := m.(*resourceCtx).Clone()
		if !model.isPublished {
			continue
		}
		if !hasMatchingType(model.resourceId.GetResourceTypes(), typeFilter) {
			continue
		}
		resources, ok := clonedModels[model.resourceId.GetDeviceId()]
		if !ok {
			resources = make(map[string]*resourceCtx)
			clonedModels[model.resourceId.GetDeviceId()] = resources
		}
		resources[model.resourceId.GetHref()] = model
	}

	return clonedModels, nil
}

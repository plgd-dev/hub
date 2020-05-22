package service

import (
	"context"
	"fmt"
	"time"

	projectionRA "github.com/go-ocf/cloud/resource-aggregate/cqrs/projection"
	"github.com/go-ocf/cqrs/eventbus"
	"github.com/go-ocf/cqrs/eventstore"
	"github.com/go-ocf/kit/strings"
	cache "github.com/patrickmn/go-cache"
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
		return nil, fmt.Errorf("cannot create server: %v", err)
	}
	cache := cache.New(expiration, expiration)
	cache.OnEvicted(func(deviceID string, _ interface{}) {
		projection.Unregister(deviceID)
	})
	return &Projection{Projection: projection, cache: cache}, nil
}

func (p *Projection) GetResourceCtxs(ctx context.Context, resourceIDsFilter, typeFilter, deviceIDs strings.Set) (map[string]map[string]*resourceCtx, error) {
	models := make([]eventstore.Model, 0, 32)

	for deviceID := range deviceIDs {
		loaded, err := p.Register(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("cannot register to projection %v", err)
		}
		if !loaded {
			defer func() {
				p.Unregister(deviceID)
			}()

		}
		p.cache.Set(deviceID, loaded, cache.DefaultExpiration)
		if len(resourceIDsFilter) > 0 {
			for resourceID := range resourceIDsFilter {
				m := p.Models(deviceID, resourceID)
				if len(m) > 0 {
					models = append(models, m...)
				}
			}
		} else {
			m := p.Models(deviceID, "")
			if len(m) > 0 {
				models = append(models, m...)
			}
		}
	}

	clonedModels := make(map[string]map[string]*resourceCtx)
	for _, m := range models {
		model := m.(*resourceCtx).Clone()
		if !model.isPublished {
			continue
		}
		if !hasMatchingType(model.resource.GetResourceTypes(), typeFilter) {
			continue
		}
		resources, ok := clonedModels[model.resource.GetDeviceId()]
		if !ok {
			resources = make(map[string]*resourceCtx)
			clonedModels[model.resource.GetDeviceId()] = resources
		}
		resources[model.resource.GetId()] = model
	}

	return clonedModels, nil
}

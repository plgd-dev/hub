package service

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	projectionRA "github.com/plgd-dev/cloud/resource-aggregate/cqrs/projection"
	"github.com/plgd-dev/cqrs/eventbus"
	"github.com/plgd-dev/cqrs/eventstore"
	"github.com/plgd-dev/kit/strings"
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
		return nil, fmt.Errorf("cannot create server: %w", err)
	}
	cache := cache.New(expiration, expiration)
	cache.OnEvicted(func(deviceID string, _ interface{}) {
		projection.Unregister(deviceID)
	})
	return &Projection{Projection: projection, cache: cache}, nil
}

func (p *Projection) getModels(ctx context.Context, deviceID, resourceID string) ([]eventstore.Model, error) {
	loaded, err := p.Register(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("cannot register to projection for device %v: %w", deviceID, err)
	}
	if loaded {
		p.cache.Set(deviceID, loaded, cache.DefaultExpiration)
	} else {
		defer func(ID string) {
			p.Unregister(ID)
		}(deviceID)
	}
	m := p.Models(deviceID, resourceID)
	if !loaded && len(m) == 0 {
		err := p.ForceUpdate(ctx, deviceID, resourceID)
		if err == nil {
			m = p.Models(deviceID, resourceID)
		}
	}
	return m, nil
}

func (p *Projection) GetResourceCtxs(ctx context.Context, resourceIDsFilter []*pb.ResourceId, typeFilter, deviceIDs strings.Set) (map[string]map[string]*resourceCtx, error) {
	models := make([]eventstore.Model, 0, 32)
	for _, res := range resourceIDsFilter {
		m, err := p.getModels(ctx, res.GetDeviceId(), res.ID())
		if err != nil {
			return nil, err
		}
		models = append(models, m...)
	}

	for deviceID := range deviceIDs {
		m, err := p.getModels(ctx, deviceID, "")
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

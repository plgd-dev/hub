package projection

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	kitSync "github.com/plgd-dev/kit/sync"
)

// Projection projects events from resource aggregate.
type Projection struct {
	cqrsProjection *projection

	topicManager *TopicManager
	refCountMap  *kitSync.Map
}

// NewProjection creates new resource projection.
func NewProjection(ctx context.Context, name string, store eventstore.EventStore, subscriber eventbus.Subscriber, factoryModel eventstore.FactoryModelFunc) (*Projection, error) {
	cqrsProjection, err := newProjection(ctx, store, name, subscriber, factoryModel, func(template string, args ...interface{}) {})
	if err != nil {
		return nil, fmt.Errorf("cannot create Projection: %w", err)
	}
	return &Projection{
		cqrsProjection: cqrsProjection,
		topicManager:   NewTopicManager(utils.GetDeviceSubject),
		refCountMap:    kitSync.NewMap(),
	}, nil
}

type deviceProjection struct {
	mutex    sync.Mutex
	released bool
	deviceID string
}

// Register registers deviceID, loads events from eventstore and subscribe to eventbus.
// It can be called multiple times for same deviceID but after successful the a call Unregister
// must be called same times to free resources.
func (p *Projection) Register(ctx context.Context, deviceID string) (created bool, err error) {
	v, loaded := p.refCountMap.LoadOrStoreWithFunc(deviceID, func(v interface{}) interface{} {
		r := v.(*kitSync.RefCounter)
		r.Acquire()
		return r
	}, func() interface{} {
		return kitSync.NewRefCounter(&deviceProjection{
			deviceID: deviceID,
		}, func(ctx context.Context, data interface{}) error {
			d := data.(*deviceProjection)
			d.released = true
			return nil
		})
	})
	r := v.(*kitSync.RefCounter)
	d := r.Data().(*deviceProjection)
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if loaded {
		return false, nil
	}
	topics, updateSubscriber := p.topicManager.Add(deviceID)
	if updateSubscriber {
		err := p.cqrsProjection.SubscribeTo(topics)
		if err != nil {
			p.release(r)
			return false, fmt.Errorf("cannot register device %v: %w", deviceID, err)
		}
	}

	err = p.cqrsProjection.Project(ctx, []eventstore.SnapshotQuery{{GroupID: deviceID}})
	if err != nil {
		p.release(r)
		return false, fmt.Errorf("cannot register device %v: %w", deviceID, err)
	}

	return true, nil
}

// Unregister unregisters device and his resource from projection.
func (p *Projection) Unregister(deviceID string) error {
	v, ok := p.refCountMap.LoadWithFunc(deviceID, func(v interface{}) interface{} {
		r := v.(*kitSync.RefCounter)
		r.Acquire()
		return r
	})
	if !ok {
		return fmt.Errorf("cannot unregister projection for %v: not found", deviceID)
	}
	r := v.(*kitSync.RefCounter)
	d := r.Data().(*deviceProjection)
	d.mutex.Lock()
	defer d.mutex.Unlock()
	p.release(r)
	return p.release(r)
}

// Models returns models for device, resource or nil for non exist.
func (p *Projection) Models(resourceID *commands.ResourceId) []eventstore.Model {
	return p.cqrsProjection.Models([]eventstore.SnapshotQuery{{GroupID: resourceID.GetDeviceId(), AggregateID: resourceID.ToUUID()}})
}

// ForceUpdate invokes update registered resource model from evenstore.
func (p *Projection) ForceUpdate(ctx context.Context, resourceID *commands.ResourceId) error {
	v, ok := p.refCountMap.LoadWithFunc(resourceID.GetDeviceId(), func(v interface{}) interface{} {
		r := v.(*kitSync.RefCounter)
		r.Acquire()
		return r
	})
	if !ok {
		return fmt.Errorf("cannot force update projection for %v: not found", resourceID.GetDeviceId())
	}
	r := v.(*kitSync.RefCounter)
	defer p.release(r)
	d := r.Data().(*deviceProjection)
	d.mutex.Lock()
	defer d.mutex.Unlock()

	err := p.cqrsProjection.Project(ctx, []eventstore.SnapshotQuery{{GroupID: resourceID.GetDeviceId(), AggregateID: resourceID.ToUUID()}})
	if err != nil {
		return fmt.Errorf("cannot force update projection for %v: %w", resourceID.GetDeviceId(), err)
	}
	return nil
}

func (p *Projection) release(v *kitSync.RefCounter) error {
	data := v.Data().(*deviceProjection)
	deviceID := data.deviceID
	p.refCountMap.ReplaceWithFunc(deviceID, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) {
		o := oldValue.(*kitSync.RefCounter)
		d := o.Data().(*deviceProjection)
		o.Release(context.Background())
		return o, d.released
	})
	if !data.released {
		return nil
	}
	p.refCountMap.Delete(deviceID)
	topics, updateSubscriber := p.topicManager.Remove(deviceID)
	if updateSubscriber {
		err := p.cqrsProjection.SubscribeTo(topics)
		if err != nil {
			log.Errorf("cannot change topics for projection device %v: %w", deviceID, err)
		}
	}
	return p.cqrsProjection.Forget([]eventstore.SnapshotQuery{
		{GroupID: deviceID},
	})
}

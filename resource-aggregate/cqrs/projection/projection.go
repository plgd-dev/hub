package projection

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

// Projection projects events from resource aggregate.
type Projection struct {
	cqrsProjection *projection

	topicManager *TopicManager
	refCountMap  *kitSync.Map
}

// NewProjection creates new resource projection.
func NewProjection(ctx context.Context, name string, store eventstore.EventStore, subscriber eventbus.Subscriber, factoryModel eventstore.FactoryModelFunc) (*Projection, error) {
	cqrsProjection, err := newProjection(ctx, store, name, subscriber, factoryModel, func(string, ...interface{}) {
		// no-op if not set
	})
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
		}, func(_ context.Context, data interface{}) error {
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
	releaseAndReturnError := func(deviceID string, err error) error {
		var errors *multierror.Error
		errors = multierror.Append(errors, fmt.Errorf("cannot register device %v: %w", deviceID, err))
		if err := p.release(r); err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot register device: %w", err))
		}
		return errors.ErrorOrNil()
	}
	if updateSubscriber {
		err = p.cqrsProjection.SubscribeTo(topics)
		if err != nil {
			return false, releaseAndReturnError(deviceID, err)
		}
	}

	err = p.cqrsProjection.Project(ctx, []eventstore.SnapshotQuery{{GroupID: deviceID}})
	if err != nil {
		return false, releaseAndReturnError(deviceID, err)
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
	var errors *multierror.Error
	for range 2 {
		if err := p.release(r); err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot unregister projection for %v: %w", deviceID, err))
		}
	}
	return errors.ErrorOrNil()
}

// Models returns models via onModel function for device, resource or nil for non exist.
func (p *Projection) Models(onModel func(eventstore.Model) (wantNext bool), resourceIDs ...*commands.ResourceId) {
	q := make([]eventstore.SnapshotQuery, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		q = append(q, eventstore.SnapshotQuery{GroupID: resourceID.GetDeviceId(), AggregateID: resourceID.ToUUID().String()})
	}
	p.cqrsProjection.Models(q, onModel)
}

func (p *Projection) GroupsModels(onModel func(eventstore.Model) (wantNext bool), groups ...string) {
	q := make([]eventstore.SnapshotQuery, 0, len(groups))
	for _, group := range groups {
		q = append(q, eventstore.SnapshotQuery{GroupID: group})
	}
	p.cqrsProjection.Models(q, onModel)
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
	defer func() {
		if err := p.release(r); err != nil {
			log.Errorf("cannot release projection: %w", err)
		}
	}()
	d := r.Data().(*deviceProjection)
	d.mutex.Lock()
	defer d.mutex.Unlock()

	err := p.cqrsProjection.Project(ctx, []eventstore.SnapshotQuery{{GroupID: resourceID.GetDeviceId(), AggregateID: resourceID.ToUUID().String()}})
	if err != nil {
		return fmt.Errorf("cannot force update projection for %v: %w", resourceID.GetDeviceId(), err)
	}
	return nil
}

func (p *Projection) release(v *kitSync.RefCounter) error {
	data := v.Data().(*deviceProjection)
	deviceID := data.deviceID
	p.refCountMap.ReplaceWithFunc(deviceID, func(oldValue interface{}, _ bool) (newValue interface{}, doDelete bool) {
		o := oldValue.(*kitSync.RefCounter)
		d := o.Data().(*deviceProjection)
		if err := o.Release(context.Background()); err != nil {
			log.Errorf("cannot release projection device %v: %w", d.deviceID, err)
		}
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

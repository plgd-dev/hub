package projection

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore"
)

// Projection project events to user defined model from evenstore and update it by events from subscriber.
type projection struct {
	//immutable
	projection *eventstore.Projection
	ctx        context.Context
	cancel     context.CancelFunc

	subscriber     eventbus.Subscriber
	subscriptionID string

	//mutable part
	lock     sync.Mutex
	observer eventbus.Observer
}

// NewProjection creates projection.
func newProjection(ctx context.Context, store eventstore.EventStore, subscriptionID string, subscriber eventbus.Subscriber, factoryModel eventstore.FactoryModelFunc, LogDebugfFunc eventstore.LogDebugfFunc) (*projection, error) {
	if store == nil {
		return nil, errors.New("invalid handle of event store")
	}

	projCtx, projCancel := context.WithCancel(ctx)

	rd := projection{
		projection:     eventstore.NewProjection(store, factoryModel, LogDebugfFunc),
		ctx:            projCtx,
		cancel:         projCancel,
		subscriber:     subscriber,
		subscriptionID: subscriptionID,
	}

	return &rd, nil
}

// Project load events from aggregates that below to path.
func (p *projection) Project(ctx context.Context, query []eventstore.SnapshotQuery) error {
	return p.projection.Project(ctx, query)
}

// Forget projection for certain query.
func (p *projection) Forget(query []eventstore.SnapshotQuery) error {
	return p.projection.Forget(query)
}

// Handle events to projection. This events comes from eventbus and it can trigger reload on eventstore.
func (p *projection) Handle(ctx context.Context, iter eventstore.Iter) error {
	return p.projection.HandleWithReload(ctx, iter)
}

// SubscribeTo set topics for observation for update events.
func (p *projection) SubscribeTo(topics []string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.subscriber == nil {
		return nil
	}
	if p.observer == nil {
		if p.subscriber == nil {
			return fmt.Errorf("projection doesn't support subscribe to topics")
		}
		observer, err := p.subscriber.Subscribe(p.ctx, p.subscriptionID, topics, p)
		if err != nil {
			return fmt.Errorf("projection cannot subscribe to topics: %w", err)
		}
		p.observer = observer
	}
	err := p.observer.SetTopics(p.ctx, topics)
	if err != nil {
		return fmt.Errorf("projection cannot set topics: %w", err)
	}

	return nil
}

// Models get models from projection
func (p *projection) Models(queries []eventstore.SnapshotQuery) []eventstore.Model {
	return p.projection.Models(queries)
}

// Close cancel projection.
func (p *projection) Close() error {
	p.cancel()
	return p.observer.Close()
}

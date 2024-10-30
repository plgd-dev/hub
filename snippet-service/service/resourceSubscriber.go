package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"go.opentelemetry.io/otel/trace"
)

type ResourceSubscriber struct {
	natsClient          *natsClient.Client
	subscriptionHandler eventbus.Handler
	subscriber          *subscriber.Subscriber
	observer            eventbus.Observer
}

func NewResourceSubscriber(ctx context.Context, config natsClient.ConfigSubscriber, subscriptionID string, fileWatcher *fsnotify.Watcher, logger log.Logger, tp trace.TracerProvider, handler eventbus.Handler) (*ResourceSubscriber, error) {
	nats, err := natsClient.New(config.Config, fileWatcher, logger, tp)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}

	subscriber, err := subscriber.New(nats.GetConn(),
		config.PendingLimits, config.LeadResourceType.IsEnabled(),
		logger,
		subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource subscriber: %w", err)
	}

	const owner = "*"
	subjectsResourceChanged := subscriber.GetResourceEventSubjects(owner, commands.NewResourceID("*", "*"), (&events.ResourceChanged{}).EventType())
	subjectsResourceUpdated := subscriber.GetResourceEventSubjects(owner, commands.NewResourceID("*", "*"), (&events.ResourceUpdated{}).EventType())
	observer, err := subscriber.Subscribe(ctx, subscriptionID, append(subjectsResourceChanged, subjectsResourceUpdated...), handler)
	if err != nil {
		subscriber.Close()
		nats.Close()
		return nil, fmt.Errorf("cannot subscribe to resource change events: %w", err)
	}

	return &ResourceSubscriber{
		natsClient:          nats,
		subscriptionHandler: handler,
		subscriber:          subscriber,
		observer:            observer,
	}, nil
}

func (r *ResourceSubscriber) Close() error {
	err := r.observer.Close()
	r.subscriber.Close()
	r.natsClient.Close()
	return err
}

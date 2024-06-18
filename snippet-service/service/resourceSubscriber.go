package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

type ResourceSubscriber struct {
	natsClient          *natsClient.Client
	subscriptionHandler eventbus.Handler
	subscriber          *subscriber.Subscriber
	observer            eventbus.Observer
}

func WithAllDevicesAndResources() func(values map[string]string) {
	return func(values map[string]string) {
		values[utils.DeviceIDKey] = "*"
		values[utils.HrefIDKey] = "*"
	}
}

func NewResourceSubscriber(ctx context.Context, config natsClient.Config, fileWatcher *fsnotify.Watcher, logger log.Logger, handler eventbus.Handler) (*ResourceSubscriber, error) {
	nats, err := natsClient.New(config, fileWatcher, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create nats client: %w", err)
	}

	subscriber, err := subscriber.New(nats.GetConn(),
		config.PendingLimits,
		logger,
		subscriber.WithUnmarshaler(utils.Unmarshal))
	if err != nil {
		nats.Close()
		return nil, fmt.Errorf("cannot create resource subscriber: %w", err)
	}

	subscriptionID := uuid.NewString()
	const owner = "*"
	subjectResourceChanged := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent,
		isEvents.WithOwner(owner),
		WithAllDevicesAndResources(),
		isEvents.WithEventType((&events.ResourceChanged{}).EventType()))
	subjectResourceUpdated := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent,
		isEvents.WithOwner(owner),
		WithAllDevicesAndResources(),
		isEvents.WithEventType((&events.ResourceUpdated{}).EventType()))
	observer, err := subscriber.Subscribe(ctx, subscriptionID, []string{subjectResourceChanged, subjectResourceUpdated}, handler)
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

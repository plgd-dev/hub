package store

import (
	"context"
	"time"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
)

type SubscriptionQuery struct {
	SubscriptionID string
	DeviceID       string
	Href           string
	Type           Type
}

type DevicesSubscriptionQuery struct {
	SubscriptionID string
	LastCheck      time.Time
}

type SubscriptionIter interface {
	Next(ctx context.Context, sub *Subscription) bool
	Err() error
}

type DevicesSubscriptionIter interface {
	Next(ctx context.Context, sub *DevicesSubscription) bool
	Err() error
}

type SubscriptionHandler interface {
	Handle(ctx context.Context, iter SubscriptionIter) (err error)
}

type DevicesSubscriptionHandler interface {
	Handle(ctx context.Context, iter DevicesSubscriptionIter) (err error)
}

type Store interface {
	SaveDevicesSubscription(ctx context.Context, sub DevicesSubscription) error
	LoadDevicesSubscriptions(ctx context.Context, query DevicesSubscriptionQuery, h DevicesSubscriptionHandler) error
	UpdateDevicesSubscription(ctx context.Context, subscriptionID string, lastDevicesRegistered events.DevicesRegistered, lastDevicesOnline events.DevicesOnline, lastDevicesOffline events.DevicesOffline) error
	PopDevicesSubscription(ctx context.Context, subscriptionID string) (DevicesSubscription, error)

	SaveSubscription(ctx context.Context, sub Subscription) error
	PopSubscription(ctx context.Context, subscriptionID string) (Subscription, error)
	LoadSubscriptions(ctx context.Context, query SubscriptionQuery, h SubscriptionHandler) error
	IncrementSubscriptionSequenceNumber(ctx context.Context, subscriptionID string) (uint64, error)
}

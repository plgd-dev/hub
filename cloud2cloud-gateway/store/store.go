package store

import (
	"context"
	"time"
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

type SubscriptionHandler interface {
	Handle(ctx context.Context, iter SubscriptionIter) (err error)
}

type Store interface {
	SaveSubscription(ctx context.Context, sub Subscription) error
	PopSubscription(ctx context.Context, subscriptionID string) (Subscription, error)
	LoadSubscriptions(ctx context.Context, query SubscriptionQuery, h SubscriptionHandler) error
	IncrementSubscriptionSequenceNumber(ctx context.Context, subscriptionID string) (uint64, error)
	SetInitialized(ctx context.Context, subscriptionID string) error
}

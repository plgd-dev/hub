package store

import (
	"context"
)

type Query struct {
	ID string
}

type LinkedAccountIter interface {
	Next(ctx context.Context, sub *LinkedAccount) bool
	Err() error
}

type LinkedAccountHandler interface {
	Handle(ctx context.Context, iter LinkedAccountIter) (err error)
}

type LinkedCloudIter interface {
	Next(ctx context.Context, sub *LinkedCloud) bool
	Err() error
}

type LinkedCloudHandler interface {
	Handle(ctx context.Context, iter LinkedCloudIter) (err error)
}

type SubscriptionQuery struct {
	SubscriptionID  string
	LinkedAccountID string
	DeviceID        string
	Href            string
	Type            Type
}

type SubscriptionIter interface {
	Next(ctx context.Context, sub *Subscription) bool
	Err() error
}

type SubscriptionHandler interface {
	Handle(ctx context.Context, iter SubscriptionIter) (err error)
}

type Store interface {
	UpdateLinkedCloud(ctx context.Context, sub LinkedCloud) error
	InsertLinkedCloud(ctx context.Context, sub LinkedCloud) error
	RemoveLinkedCloud(ctx context.Context, ConfigId string) error
	LoadLinkedClouds(ctx context.Context, query Query, h LinkedCloudHandler) error

	UpdateLinkedAccount(ctx context.Context, sub LinkedAccount) error
	InsertLinkedAccount(ctx context.Context, sub LinkedAccount) error
	RemoveLinkedAccount(ctx context.Context, LinkedAccountId string) error
	LoadLinkedAccounts(ctx context.Context, query Query, h LinkedAccountHandler) error

	LoadSubscriptions(ctx context.Context, query []SubscriptionQuery, h SubscriptionHandler) error
	FindOrCreateSubscription(ctx context.Context, sub Subscription) (Subscription, error)
	RemoveSubscriptions(ctx context.Context, query SubscriptionQuery) error
}

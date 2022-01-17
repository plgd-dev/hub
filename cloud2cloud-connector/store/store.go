package store

import (
	"context"
)

type Query struct {
	ID            string
	LinkedCloudID string
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

type Store interface {
	UpdateLinkedCloud(ctx context.Context, sub LinkedCloud) error
	InsertLinkedCloud(ctx context.Context, sub LinkedCloud) error
	RemoveLinkedCloud(ctx context.Context, ConfigID string) error
	LoadLinkedClouds(ctx context.Context, query Query, h LinkedCloudHandler) error

	UpdateLinkedAccount(ctx context.Context, sub LinkedAccount) error
	InsertLinkedAccount(ctx context.Context, sub LinkedAccount) error
	RemoveLinkedAccount(ctx context.Context, LinkedAccountID string) error
	LoadLinkedAccounts(ctx context.Context, query Query, h LinkedAccountHandler) error

	Close(ctx context.Context) error
}

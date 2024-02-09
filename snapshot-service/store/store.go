package store

import (
	"context"
	"errors"
)

var ErrNotSupported = errors.New("not supported")

type Store interface {
	Close(ctx context.Context) error
}

package store

import (
	"context"
	"errors"
	"time"

	"github.com/plgd-dev/hub/v2/integration-service/pb"
)

var ErrNotSupported = errors.New("not supported")

type ConfigurationRecord = pb.Configuration

type Store interface {
	CreateRecord(ctx context.Context, r *ConfigurationRecord) error

	DeleteExpiredRecords(ctx context.Context, now time.Time) (int64, error)

	Close(ctx context.Context) error
}

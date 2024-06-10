package store

import (
	"context"
	"errors"
	"time"

	"github.com/plgd-dev/hub/v2/integration-service/pb"
)

var ErrNotSupported = errors.New("not supported")

type (
	ConfigurationRecord     = pb.Configuration
	GetConfigurationRequest = pb.GetConfigurationRequest
)

type Store interface {
	CreateRecord(ctx context.Context, r *ConfigurationRecord) error

	DeleteExpiredRecords(ctx context.Context, now time.Time) (int64, error)

	GetRecord(ctx context.Context, confID string, query *GetConfigurationRequest, rec *ConfigurationRecord) error

	Close(ctx context.Context) error
}

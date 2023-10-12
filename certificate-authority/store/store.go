package store

import (
	"context"
	"errors"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
)

var ErrNotSupported = errors.New("not supported")

type (
	SigningRecordsQuery       = pb.GetSigningRecordsRequest
	DeleteSigningRecordsQuery = pb.DeleteSigningRecordsRequest
)

type SigningRecordIter interface {
	Next(ctx context.Context, SigningRecord *SigningRecord) bool
	Err() error
}

type (
	LoadSigningRecordsFunc = func(ctx context.Context, iter SigningRecordIter) (err error)
)

type Store interface {
	// CreateSigningRecord creates a new signing record. If the record already exists, it will throw an error.
	CreateSigningRecord(ctx context.Context, record *SigningRecord) error
	// UpdateSigningRecord updates an existing signing record. If the record does not exist, it will create a new one.
	UpdateSigningRecord(ctx context.Context, record *SigningRecord) error
	DeleteSigningRecords(ctx context.Context, ownerID string, query *DeleteSigningRecordsQuery) (int64, error)
	LoadSigningRecords(ctx context.Context, ownerID string, query *SigningRecordsQuery, h LoadSigningRecordsFunc) error

	// DeleteNonDeviceExpiredRecords deletes all expired records that are not associated with a device.
	// For CqlDB, this is a no-op because expired records are deleted by Cassandra automatically.
	DeleteNonDeviceExpiredRecords(ctx context.Context, now time.Time) (int64, error)

	Close(ctx context.Context) error
}

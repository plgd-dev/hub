package store

import (
	"context"

	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
)

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
	UpdateSigningRecord(ctx context.Context, sub *SigningRecord) error
	DeleteSigningRecords(ctx context.Context, ownerID string, query *DeleteSigningRecordsQuery) error
	LoadSigningRecords(ctx context.Context, ownerID string, query *SigningRecordsQuery, h LoadSigningRecordsFunc) error

	Close(ctx context.Context) error
}

package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
)

var (
	ErrNotSupported = errors.New("not supported")
	ErrNotFound     = errors.New("no document found")
	ErrDuplicateID  = errors.New("duplicate ID")
)

type (
	Process[T any] func(v *T) error

	SigningRecordsQuery       = pb.GetSigningRecordsRequest
	DeleteSigningRecordsQuery = pb.DeleteSigningRecordsRequest
	RevokeSigningRecordsQuery = pb.DeleteSigningRecordsRequest

	UpdateRevocationListQuery struct {
		IssuerID            string
		IssuedAt            int64 // 0 is allowed, the timestamp will be generated when the CRL is first issued
		ValidUntil          int64 // 0 is allowed, the timestamp will be generated when the CRL is first issued
		UpdateIfExpired     bool
		RevokedCertificates []*RevocationListCertificate
	}
)

type Store interface {
	// CreateSigningRecord creates a new signing record. If the record already exists, it will throw an error.
	CreateSigningRecord(ctx context.Context, record *SigningRecord) error
	// UpdateSigningRecord updates an existing signing record. If the record does not exist, it will create a new one.
	UpdateSigningRecord(ctx context.Context, record *SigningRecord) error
	DeleteSigningRecords(ctx context.Context, ownerID string, query *DeleteSigningRecordsQuery) (int64, error)
	LoadSigningRecords(ctx context.Context, ownerID string, query *SigningRecordsQuery, p Process[SigningRecord]) error

	// DeleteNonDeviceExpiredRecords deletes all expired records that are not associated with a device.
	// For CqlDB, this is a no-op because expired records are deleted by Cassandra automatically.
	DeleteNonDeviceExpiredRecords(ctx context.Context, now time.Time) (int64, error)

	// Check if the implementation supports the RevocationList feature
	SupportsRevocationList() bool
	// InsertRevocationLists adds revocations lists to the database
	InsertRevocationLists(ctx context.Context, rls ...*RevocationList) error
	// UpdateRevocationList updates revocation list number and validity and adds certificates to revocation list. Returns the updated revocation list.
	UpdateRevocationList(ctx context.Context, query *UpdateRevocationListQuery) (*RevocationList, error)
	// Get valid latest issued or issue a new one revocation list
	GetLatestIssuedOrIssueRevocationList(ctx context.Context, issuerID string, validFor time.Duration) (*RevocationList, error)

	// Removed matched signing records and move them to a revocation list.
	RevokeSigningRecords(ctx context.Context, ownerID string, query *RevokeSigningRecordsQuery) (int64, error)

	Close(ctx context.Context) error
}

func (q *UpdateRevocationListQuery) Validate() error {
	if _, err := uuid.Parse(q.IssuerID); err != nil {
		return fmt.Errorf("invalid revocation list issuerID(%v): %w", q.IssuerID, err)
	}
	return nil
}

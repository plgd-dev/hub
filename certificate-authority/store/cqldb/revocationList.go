package cqldb

import (
	"context"
	"time"

	"github.com/plgd-dev/hub/v2/certificate-authority/store"
)

func (s *Store) SupportsRevocationList() bool {
	return false
}

func (s *Store) InsertRevocationLists(context.Context, ...*store.RevocationList) error {
	return store.ErrNotSupported
}

func (s *Store) UpdateRevocationList(context.Context, *store.UpdateRevocationListQuery) (*store.RevocationList, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetRevocationList(context.Context, string, bool) (*store.RevocationList, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetLatestIssuedOrIssueRevocationList(context.Context, string, time.Duration) (*store.RevocationList, error) {
	return nil, store.ErrNotSupported
}

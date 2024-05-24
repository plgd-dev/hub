package cqldb

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// Store implements an Store for cqldb.
type Store struct {
	table  string
	client *Client
	logger log.Logger
}

func NewStore(table string, client *Client, logger log.Logger) *Store {
	return &Store{
		table:  table,
		client: client,
		logger: logger,
	}
}

func (s *Store) Table() string {
	return s.client.Keyspace() + "." + s.table
}

func (s *Store) Session() *gocql.Session {
	return s.client.Session()
}

func (s *Store) AddCloseFunc(f func()) {
	s.client.AddCloseFunc(f)
}

func (s *Store) Clear(ctx context.Context) error {
	err := s.client.DropKeyspace(ctx)
	if err != nil {
		return fmt.Errorf("cannot clear: %w", err)
	}
	return nil
}

// Clear documents in collections, but don't drop the database or the collections
func (s *Store) ClearTable(ctx context.Context) error {
	return s.client.Session().Query("truncate " + s.Table() + ";").WithContext(ctx).Exec()
}

// Close closes the database session.
func (s *Store) Close(_ context.Context) error {
	s.client.Close()
	return nil
}

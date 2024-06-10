package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateCondition(context.Context, *pb.Condition) (*pb.Condition, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) UpdateCondition(context.Context, *pb.Condition) (*pb.Condition, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetConditions(context.Context, string, *pb.GetConditionsRequest, store.ProcessConditions) error {
	return store.ErrNotSupported
}

func (s *Store) DeleteConditions(context.Context, string, *pb.DeleteConditionsRequest) (int64, error) {
	return 0, store.ErrNotSupported
}

func (s *Store) InsertConditions(context.Context, ...*store.Condition) error {
	return store.ErrNotSupported
}

func (s *Store) GetLatestConditions(context.Context, string, *store.GetLatestConditionsQuery, store.ProcessConditions) error {
	return store.ErrNotSupported
}

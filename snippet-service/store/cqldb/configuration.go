package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateConfiguration(context.Context, *pb.Configuration) (*pb.Configuration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) UpdateConfiguration(context.Context, *pb.Configuration) (*pb.Configuration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) LoadConfigurations(context.Context, string, *pb.GetConfigurationsRequest, store.LoadConfigurationsFunc) error {
	return store.ErrNotSupported
}

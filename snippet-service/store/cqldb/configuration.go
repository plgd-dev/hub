package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateConfiguration(context.Context, *pb.Configuration) (*pb.Configuration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) DeleteConfigurations(context.Context, string, *pb.DeleteConfigurationsRequest) (int64, error) {
	return 0, store.ErrNotSupported
}

func (s *Store) GetConfigurations(context.Context, string, *pb.GetConfigurationsRequest, store.GetConfigurationsFunc) error {
	return store.ErrNotSupported
}

func (s *Store) UpdateConfiguration(context.Context, *pb.Configuration) (*pb.Configuration, error) {
	return nil, store.ErrNotSupported
}

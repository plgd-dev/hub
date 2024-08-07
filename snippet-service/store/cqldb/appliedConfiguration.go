package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) GetAppliedConfigurations(context.Context, string, *pb.GetAppliedConfigurationsRequest, store.ProccessAppliedConfigurations) error {
	return store.ErrNotSupported
}

func (s *Store) DeleteAppliedConfigurations(context.Context, string, *pb.DeleteAppliedConfigurationsRequest) (int64, error) {
	return 0, store.ErrNotSupported
}

func (s *Store) CreateAppliedConfiguration(context.Context, *pb.AppliedConfiguration, bool) (*pb.AppliedConfiguration, *pb.AppliedConfiguration, error) {
	return nil, nil, store.ErrNotSupported
}

func (s *Store) InsertAppliedConfigurations(context.Context, ...*store.AppliedConfiguration) error {
	return store.ErrNotSupported
}

func (s *Store) UpdateAppliedConfigurationResource(context.Context, string, store.UpdateAppliedConfigurationResourceRequest) (*pb.AppliedConfiguration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetPendingAppliedConfigurationResourceUpdates(context.Context, bool, store.ProccessAppliedConfigurations) (int64, error) {
	return 0, store.ErrNotSupported
}

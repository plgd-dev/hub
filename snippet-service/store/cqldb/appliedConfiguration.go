package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) GetAppliedConfigurations(context.Context, string, *pb.GetAppliedDeviceConfigurationsRequest, store.ProccessAppliedDeviceConfigurations) error {
	return store.ErrNotSupported
}

func (s *Store) DeleteAppliedConfigurations(context.Context, string, *pb.DeleteAppliedDeviceConfigurationsRequest) (int64, error) {
	return 0, store.ErrNotSupported
}

func (s *Store) CreateAppliedConfiguration(context.Context, *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) InsertAppliedConfigurations(context.Context, ...*store.AppliedDeviceConfiguration) error {
	return store.ErrNotSupported
}

func (s *Store) UpdateAppliedConfiguration(context.Context, *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) UpdateAppliedConfigurationPendingResources(context.Context, ...*store.UpdateAppliedConfigurationPendingResourceRequest) error {
	return store.ErrNotSupported
}

package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateAppliedDeviceConfiguration(context.Context, *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	return nil, store.ErrNotSupported
}

func (s *Store) GetAppliedDeviceConfigurations(context.Context, string, *pb.GetAppliedDeviceConfigurationsRequest, store.ProccessAppliedDeviceConfigurations) error {
	return store.ErrNotSupported
}

func (s *Store) InsertAppliedConfigurations(context.Context, ...*store.AppliedDeviceConfiguration) error {
	return store.ErrNotSupported
}

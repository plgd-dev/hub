package cqldb

import (
	"context"

	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateAppliedDeviceConfiguration(context.Context, *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	return nil, store.ErrNotSupported
}

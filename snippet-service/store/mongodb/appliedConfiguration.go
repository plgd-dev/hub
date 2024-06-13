package mongodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
)

func (s *Store) CreateAppliedDeviceConfiguration(ctx context.Context, adc *pb.AppliedDeviceConfiguration) (*pb.AppliedDeviceConfiguration, error) {
	err := store.ValidateAppliedConfiguration(adc, false)
	if err != nil {
		return nil, err
	}
	newAdc := adc.Clone()
	if newAdc.GetId() == "" {
		newAdc.Id = uuid.NewString()
	}
	newAdc.Timestamp = time.Now().UnixNano()
	_, err = s.Collection(appliedConfigurationsCol).InsertOne(ctx, newAdc)
	if err != nil {
		return nil, err
	}
	return newAdc, nil
}

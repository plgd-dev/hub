package store

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

func ValidateAppliedConfiguration(c *pb.AppliedDeviceConfiguration, isUpdate bool) error {
	if err := c.Validate(isUpdate); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

type UpdateAppliedConfigurationResourceRequest struct {
	AppliedConfigurationID string
	AppliedCondition       *pb.AppliedDeviceConfiguration_RelationTo
	StatusFilter           []pb.AppliedDeviceConfiguration_Resource_Status
	Resource               *pb.AppliedDeviceConfiguration_Resource
}

func (u *UpdateAppliedConfigurationResourceRequest) Validate() error {
	if _, err := uuid.Parse(u.AppliedConfigurationID); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid ID(%v): %w", u.AppliedConfigurationID, err))
	}
	if u.Resource == nil {
		return errInvalidArgument(errors.New("resource is required"))
	}
	if err := u.Resource.Validate(); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

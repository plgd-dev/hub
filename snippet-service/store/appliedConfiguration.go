package store

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

type AppliedDeviceConfiguration = pb.AppliedDeviceConfiguration

func ValidateAppliedConfiguration(c *pb.AppliedDeviceConfiguration, isUpdate bool) error {
	if err := c.Validate(isUpdate); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

type UpdateAppliedConfigurationPendingResourceRequest struct {
	ID     string
	Owner  string
	Href   string
	Status pb.AppliedDeviceConfiguration_Resource_Status
}

func (u *UpdateAppliedConfigurationPendingResourceRequest) Validate() error {
	if _, err := uuid.Parse(u.ID); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid ID(%v): %w", u.ID, err))
	}
	if u.Status != pb.AppliedDeviceConfiguration_Resource_DONE &&
		u.Status != pb.AppliedDeviceConfiguration_Resource_TIMEOUT {
		return errInvalidArgument(fmt.Errorf("invalid status(%v)", u.Status.String()))
	}
	return nil
}

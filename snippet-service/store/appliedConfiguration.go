package store

import (
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

type AppliedDeviceConfiguration = pb.AppliedDeviceConfiguration

func ValidateAppliedConfiguration(c *pb.AppliedDeviceConfiguration, isUpdate bool) error {
	if err := c.Validate(isUpdate); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

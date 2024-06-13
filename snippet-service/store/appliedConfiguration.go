package store

import (
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
)

// TODO: duplicate the pb.AppliedDeviceConfiguration, but add omitempty tags to the fields
type AppliedDeviceConfiguration = pb.AppliedDeviceConfiguration

func ValidateAndNormalizeAppliedConfiguration(c *pb.AppliedDeviceConfiguration, isUpdate bool) (*pb.AppliedDeviceConfiguration, error) {
	if err := c.Validate(isUpdate); err != nil {
		return nil, errInvalidArgument(err)
	}
	return c.Clone(), nil
}

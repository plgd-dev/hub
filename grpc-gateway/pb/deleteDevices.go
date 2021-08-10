package pb

import (
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/kit/strings"
)

func (req *DeleteDevicesRequest) ToRACommand() (*commands.DeleteDevicesRequest, error) {
	deviceIds := make(strings.Set)
	deviceIds.Add(req.DeviceIdFilter...)
	delete(deviceIds, "")

	return &commands.DeleteDevicesRequest{
		DeviceIds: deviceIds.ToSlice(),
	}, nil
}

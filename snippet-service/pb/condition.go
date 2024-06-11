package pb

import "slices"

func (c *Condition) Clone() *Condition {
	return &Condition{
		Id:                 c.GetId(),
		Name:               c.GetName(),
		Enabled:            c.GetEnabled(),
		Owner:              c.GetOwner(),
		ConfigurationId:    c.GetConfigurationId(),
		ApiAccessToken:     c.GetApiAccessToken(),
		Timestamp:          c.GetTimestamp(),
		Version:            c.GetVersion(),
		DeviceIdFilter:     slices.Clone(c.GetDeviceIdFilter()),
		ResourceTypeFilter: slices.Clone(c.GetResourceTypeFilter()),
		ResourceHrefFilter: slices.Clone(c.GetResourceHrefFilter()),
		JqExpressionFilter: c.GetJqExpressionFilter(),
	}
}

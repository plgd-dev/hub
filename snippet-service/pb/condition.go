package pb

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
		DeviceIdFilter:     c.GetDeviceIdFilter(),
		ResourceTypeFilter: c.GetResourceTypeFilter(),
		ResourceHrefFilter: c.GetResourceHrefFilter(),
		JqExpressionFilter: c.GetJqExpressionFilter(),
	}
}

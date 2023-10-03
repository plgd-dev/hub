package pb

func (c *SubscribeToEvents_CreateSubscription) ConvertHTTPResourceIDFilter() []*ResourceIdFilter {
	return ResourceIdFilterFromString(c.GetHttpResourceIdFilter())
}

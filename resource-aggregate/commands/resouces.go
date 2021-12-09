package commands

func (e *EndpointInformation) Clone() *EndpointInformation {
	if e == nil {
		return nil
	}

	return &EndpointInformation{
		Endpoint: e.GetEndpoint(),
		Priority: e.GetPriority(),
	}
}

func (eis EndpointInformations) Clone() EndpointInformations {
	if eis == nil {
		return nil
	}

	e := make([]*EndpointInformation, len(eis))
	for i := range eis {
		e[i] = eis[i].Clone()
	}
	return e
}

func (r *Resource) Clone() *Resource {
	if r == nil {
		return nil
	}

	return &Resource{
		Href:                  r.GetHref(),
		DeviceId:              r.GetDeviceId(),
		ResourceTypes:         append([]string{}, r.GetResourceTypes()...),
		Interfaces:            append([]string{}, r.GetInterfaces()...),
		Anchor:                r.GetAnchor(),
		Title:                 r.GetTitle(),
		SupportedContentTypes: append([]string{}, r.GetSupportedContentTypes()...),
		ValidUntil:            r.GetValidUntil(),
		Policy: &Policy{
			BitFlags: r.GetPolicy().GetBitFlags(),
		},
		EndpointInformations: EndpointInformations(r.GetEndpointInformations()).Clone(),
	}
}

func CloneResourcesMap(resources map[string]*Resource) map[string]*Resource {
	c := make(map[string]*Resource, len(resources))
	for k, v := range resources {
		c[k] = v.Clone()
	}
	return c
}

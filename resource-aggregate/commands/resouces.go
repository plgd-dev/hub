package commands

import "github.com/plgd-dev/device/v2/schema"

func (c *Content) Clone() *Content {
	if c == nil {
		return nil
	}

	return &Content{
		Data:              c.GetData(),
		ContentType:       c.GetContentType(),
		CoapContentFormat: c.GetCoapContentFormat(),
	}
}

func (r *ResourceId) Clone() *ResourceId {
	if r == nil {
		return nil
	}

	return &ResourceId{
		DeviceId: r.GetDeviceId(),
		Href:     r.GetHref(),
	}
}

func (e *EndpointInformation) Clone() *EndpointInformation {
	if e == nil {
		return nil
	}

	return &EndpointInformation{
		Endpoint: e.GetEndpoint(),
		Priority: e.GetPriority(),
	}
}

func (e EndpointInformations) Clone() EndpointInformations {
	if e == nil {
		return nil
	}

	eis := make([]*EndpointInformation, len(e))
	for i := range e {
		eis[i] = e[i].Clone()
	}
	return eis
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

func ToPolicyBitFlags(bm schema.BitMask) int32 {
	return int32(bm)
}

package pb

import (
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/sdk/schema"
)

func (e EndpointInformation) ToRAProto() *commands.EndpointInformation {
	return &commands.EndpointInformation{
		Endpoint: e.GetEndpoint(),
		Priority: e.GetPriority(),
	}
}

func (e EndpointInformation) ToSchema() schema.Endpoint {
	return schema.Endpoint{
		URI:      e.GetEndpoint(),
		Priority: uint64(e.GetPriority()),
	}
}

type EndpointInformations []*EndpointInformation

func (e EndpointInformations) ToRAProto() []*commands.EndpointInformation {
	if e == nil {
		return nil
	}
	r := make([]*commands.EndpointInformation, 0, len(e))
	for _, v := range e {
		r = append(r, v.ToRAProto())
	}
	return r
}

func (e EndpointInformations) ToSchema() []schema.Endpoint {
	if e == nil {
		return nil
	}
	r := make([]schema.Endpoint, 0, len(e))
	for _, v := range e {
		r = append(r, v.ToSchema())
	}
	return r
}

func (p *Policies) ToRAProto() *commands.Policies {
	if p == nil {
		return nil
	}
	return &commands.Policies{
		BitFlags: p.GetBitFlags(),
	}
}

func (p *Policies) ToSchema() *schema.Policy {
	if p == nil {
		return nil
	}
	return &schema.Policy{
		BitMask: schema.BitMask(p.GetBitFlags()),
	}
}

func (l ResourceLink) ToRAProto() commands.Resource {
	return commands.Resource{
		Anchor:                l.GetAnchor(),
		DeviceId:              l.GetDeviceId(),
		EndpointInformations:  EndpointInformations(l.GetEndpointInformations()).ToRAProto(),
		Href:                  l.GetHref(),
		Interfaces:            l.GetInterfaces(),
		Policies:              l.GetPolicies().ToRAProto(),
		ResourceTypes:         l.GetTypes(),
		SupportedContentTypes: l.GetSupportedContentTypes(),
		Title:                 l.GetTitle(),
	}
}

func (l ResourceLink) ToSchema() schema.ResourceLink {
	id := (&ResourceId{DeviceId: l.GetDeviceId(), Href: l.GetHref()}).ToUUID()
	return schema.ResourceLink{
		ID:                    id,
		Anchor:                l.GetAnchor(),
		DeviceID:              l.GetDeviceId(),
		Endpoints:             EndpointInformations(l.GetEndpointInformations()).ToSchema(),
		Href:                  l.GetHref(),
		Interfaces:            l.GetInterfaces(),
		Policy:                l.GetPolicies().ToSchema(),
		ResourceTypes:         l.GetTypes(),
		SupportedContentTypes: l.GetSupportedContentTypes(),
		Title:                 l.GetTitle(),
	}
}

func RAEndpointInformationsToProto(e []*commands.EndpointInformation) []*EndpointInformation {
	if e == nil {
		return nil
	}
	r := make([]*EndpointInformation, 0, len(e))
	for _, v := range e {
		r = append(r, &EndpointInformation{
			Endpoint: v.GetEndpoint(),
			Priority: v.GetPriority(),
		})
	}
	return r
}

func RAResourceToProto(ra *commands.Resource) ResourceLink {
	var p *Policies
	if ra.Policies != nil {
		p = &Policies{
			BitFlags: ra.Policies.GetBitFlags(),
		}
	}
	return ResourceLink{
		Anchor:                ra.Anchor,
		DeviceId:              ra.DeviceId,
		EndpointInformations:  RAEndpointInformationsToProto(ra.EndpointInformations),
		Href:                  ra.Href,
		Interfaces:            ra.Interfaces,
		Policies:              p,
		Types:                 ra.ResourceTypes,
		SupportedContentTypes: ra.SupportedContentTypes,
		Title:                 ra.Title,
	}
}

func SchemaEndpointsToProto(ra []schema.Endpoint) []*EndpointInformation {
	if ra == nil {
		return nil
	}
	r := make([]*EndpointInformation, 0, len(ra))
	for _, e := range ra {
		r = append(r, &EndpointInformation{
			Endpoint: e.URI,
			Priority: int64(e.Priority),
		})
	}
	return r
}

func SchemaPolicyToProto(ra *schema.Policy) *Policies {
	if ra == nil {
		return nil
	}
	return &Policies{
		BitFlags: int32(ra.BitMask),
	}
}

func SchemaResourceLinkToProto(ra schema.ResourceLink) ResourceLink {
	return ResourceLink{
		Anchor:                ra.Anchor,
		DeviceId:              ra.DeviceID,
		EndpointInformations:  SchemaEndpointsToProto(ra.Endpoints),
		Href:                  ra.Href,
		Interfaces:            ra.Interfaces,
		Policies:              SchemaPolicyToProto(ra.Policy),
		Types:                 ra.ResourceTypes,
		SupportedContentTypes: ra.SupportedContentTypes,
		Title:                 ra.Title,
	}
}

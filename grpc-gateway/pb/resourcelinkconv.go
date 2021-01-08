package pb

import (
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/sdk/schema"
)

func (e EndpointInformation) ToRAProto() *pbRA.EndpointInformation {
	return &pbRA.EndpointInformation{
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

func (e EndpointInformations) ToRAProto() []*pbRA.EndpointInformation {
	if e == nil {
		return nil
	}
	r := make([]*pbRA.EndpointInformation, 0, len(e))
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

func (p *Policies) ToRAProto() *pbRA.Policies {
	if p == nil {
		return nil
	}
	return &pbRA.Policies{
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

func (l ResourceLink) ToRAProto() pbRA.Resource {
	return pbRA.Resource{
		Anchor:               l.GetAnchor(),
		DeviceId:             l.GetDeviceId(),
		EndpointInformations: EndpointInformations(l.GetEndpointInformations()).ToRAProto(),
		Href:                 l.GetHref(),
		Id:                   utils.MakeResourceId(l.GetDeviceId(), l.GetHref()),
		// InstanceId:            l.GetInstanceId(),
		Interfaces:            l.GetInterfaces(),
		Policies:              l.GetPolicies().ToRAProto(),
		ResourceTypes:         l.GetTypes(),
		SupportedContentTypes: l.GetSupportedContentTypes(),
		Title:                 l.GetTitle(),
	}
}

func (l ResourceLink) ToSchema() schema.ResourceLink {
	return schema.ResourceLink{
		ID:        utils.MakeResourceId(l.GetDeviceId(), l.GetHref()),
		Anchor:    l.GetAnchor(),
		DeviceID:  l.GetDeviceId(),
		Endpoints: EndpointInformations(l.GetEndpointInformations()).ToSchema(),
		Href:      l.GetHref(),
		// InstanceID:            l.GetInstanceId(),
		Interfaces:            l.GetInterfaces(),
		Policy:                l.GetPolicies().ToSchema(),
		ResourceTypes:         l.GetTypes(),
		SupportedContentTypes: l.GetSupportedContentTypes(),
		Title:                 l.GetTitle(),
	}
}

func RAEndpointInformationsToProto(e []*pbRA.EndpointInformation) []*EndpointInformation {
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

func RAResourceToProto(ra *pbRA.Resource) ResourceLink {
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

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
	id := (&commands.ResourceId{DeviceId: l.GetDeviceId(), Href: l.GetHref()}).ToUUID()
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

func RAResourceToProto(res *commands.Resource) *ResourceLink {
	var p *Policies
	if res.Policies != nil {
		p = &Policies{
			BitFlags: res.Policies.GetBitFlags(),
		}
	}
	return &ResourceLink{
		Anchor:                res.Anchor,
		DeviceId:              res.DeviceId,
		EndpointInformations:  RAEndpointInformationsToProto(res.EndpointInformations),
		Href:                  res.Href,
		Interfaces:            res.Interfaces,
		Policies:              p,
		Types:                 res.ResourceTypes,
		SupportedContentTypes: res.SupportedContentTypes,
		Title:                 res.Title,
	}
}

func RAResourcesToProto(resources map[string]*commands.Resource) []*ResourceLink {
	var resourceLinks = make([]*ResourceLink, 0, len(resources))
	for _, res := range resources {
		resourceLinks = append(resourceLinks, RAResourceToProto(res))
	}
	return resourceLinks
}

func SchemaEndpointsToProto(ra []schema.Endpoint) []*commands.EndpointInformation {
	if ra == nil {
		return nil
	}
	r := make([]*commands.EndpointInformation, 0, len(ra))
	for _, e := range ra {
		r = append(r, &commands.EndpointInformation{
			Endpoint: e.URI,
			Priority: int64(e.Priority),
		})
	}
	return r
}

func SchemaPolicyToProto(ra *schema.Policy) *commands.Policies {
	if ra == nil {
		return nil
	}
	return &commands.Policies{
		BitFlags: int32(ra.BitMask),
	}
}

func SchemaResourceLinkToRAResource(link schema.ResourceLink, ttl int32) *commands.Resource {
	return &commands.Resource{
		Href:                  link.Href,
		DeviceId:              link.DeviceID,
		ResourceTypes:         link.ResourceTypes,
		Interfaces:            link.Interfaces,
		Anchor:                link.Anchor,
		Title:                 link.Title,
		SupportedContentTypes: link.SupportedContentTypes,
		TimeToLive:            ttl,
		Policies:              SchemaPolicyToProto(link.Policy),
		EndpointInformations:  SchemaEndpointsToProto(link.Endpoints),
	}
}

func SchemaResourceLinksToRAResources(links schema.ResourceLinks, ttl int32) []*commands.Resource {
	var resources = make([]*commands.Resource, 0, len(links))
	for _, link := range links {
		resources = append(resources, SchemaResourceLinkToRAResource(link, ttl))
	}
	return resources
}

package commands

import (
	"time"

	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/sdk/schema"
)

func (e *EndpointInformation) ToSchema() schema.Endpoint {
	return schema.Endpoint{
		URI:      e.GetEndpoint(),
		Priority: uint64(e.GetPriority()),
	}
}

type EndpointInformations []*EndpointInformation

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

func (p *Policies) ToSchema() *schema.Policy {
	if p == nil {
		return nil
	}
	return &schema.Policy{
		BitMask: schema.BitMask(p.GetBitFlags()),
	}
}

func (l *Resource) ToSchema() schema.ResourceLink {
	return schema.ResourceLink{
		ID:                    l.ToUUID(),
		Anchor:                l.GetAnchor(),
		DeviceID:              l.GetDeviceId(),
		Endpoints:             EndpointInformations(l.GetEndpointInformations()).ToSchema(),
		Href:                  l.GetHref(),
		Interfaces:            l.GetInterfaces(),
		Policy:                l.GetPolicies().ToSchema(),
		ResourceTypes:         l.GetResourceTypes(),
		SupportedContentTypes: l.GetSupportedContentTypes(),
		Title:                 l.GetTitle(),
	}
}

func ResourcesToResourceLinks(resources []*Resource) []schema.ResourceLink {
	links := make([]schema.ResourceLink, 0, len(resources))
	for _, r := range resources {
		links = append(links, r.ToSchema())
	}
	return links
}

func SchemaEndpointsToRAEndpointInformations(ra []schema.Endpoint) []*EndpointInformation {
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

func SchemaPolicyToRAPolicies(ra *schema.Policy) *Policies {
	if ra == nil {
		return nil
	}
	return &Policies{
		BitFlags: int32(ra.BitMask),
	}
}

func SchemaResourceLinkToResource(link schema.ResourceLink, validUntil time.Time) *Resource {
	return &Resource{
		Href:                  link.Href,
		DeviceId:              link.DeviceID,
		ResourceTypes:         link.ResourceTypes,
		Interfaces:            link.Interfaces,
		Anchor:                link.Anchor,
		Title:                 link.Title,
		SupportedContentTypes: link.SupportedContentTypes,
		ValidUntil:            pkgTime.UnixNano(validUntil),
		Policies:              SchemaPolicyToRAPolicies(link.Policy),
		EndpointInformations:  SchemaEndpointsToRAEndpointInformations(link.Endpoints),
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

func SchemaResourceLinksToResources(links schema.ResourceLinks, validUntil time.Time) []*Resource {
	var resources = make([]*Resource, 0, len(links))
	for _, link := range links {
		resources = append(resources, SchemaResourceLinkToResource(link, validUntil))
	}
	return resources
}

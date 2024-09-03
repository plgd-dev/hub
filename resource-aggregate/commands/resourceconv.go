package commands

import (
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/internal/math"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
)

func (e *EndpointInformation) ToSchema() schema.Endpoint {
	return schema.Endpoint{
		URI:      e.GetEndpoint(),
		Priority: e.GetPriority(),
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

func (p *Policy) ToSchema() *schema.Policy {
	if p == nil {
		return nil
	}
	return &schema.Policy{
		BitMask: math.CastTo[schema.BitMask](p.GetBitFlags()),
	}
}

func (r *Resource) ToSchema() schema.ResourceLink {
	return schema.ResourceLink{
		ID:                    r.ToUUID().String(),
		Anchor:                r.GetAnchor(),
		DeviceID:              r.GetDeviceId(),
		Endpoints:             EndpointInformations(r.GetEndpointInformations()).ToSchema(),
		Href:                  r.GetHref(),
		Interfaces:            r.GetInterfaces(),
		Policy:                r.GetPolicy().ToSchema(),
		ResourceTypes:         r.GetResourceTypes(),
		SupportedContentTypes: r.GetSupportedContentTypes(),
		Title:                 r.GetTitle(),
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
			Priority: e.Priority,
		})
	}
	return r
}

func SchemaPolicyToRAPolicy(ra *schema.Policy) *Policy {
	if ra == nil {
		return nil
	}
	return &Policy{
		BitFlags: ToPolicyBitFlags(ra.BitMask),
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
		Policy:                SchemaPolicyToRAPolicy(link.Policy),
		EndpointInformations:  SchemaEndpointsToRAEndpointInformations(link.Endpoints),
	}
}

func SchemaResourceLinkToResourceId(link schema.ResourceLink) *ResourceId {
	return &ResourceId{
		Href:     link.Href,
		DeviceId: link.DeviceID,
	}
}

func SchemaResourceLinksToResources(links schema.ResourceLinks, validUntil time.Time) []*Resource {
	resources := make([]*Resource, 0, len(links))
	for _, link := range links {
		resources = append(resources, SchemaResourceLinkToResource(link, validUntil))
	}
	return resources
}

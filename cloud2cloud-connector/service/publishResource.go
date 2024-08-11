package service

import (
	"context"

	"github.com/plgd-dev/device/v2/schema"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
)

func publishResource(ctx context.Context, raClient raService.ResourceAggregateClient, link schema.ResourceLink, cmdMetadata *commands.CommandMetadata) error {
	endpoints := make([]*commands.EndpointInformation, 0, 4)
	for _, endpoint := range link.GetEndpoints() {
		endpoints = append(endpoints, &commands.EndpointInformation{
			Endpoint: endpoint.URI,
			Priority: int64(endpoint.Priority),
		})
	}
	href := pkgHttpUri.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
	_, err := raClient.PublishResourceLinks(ctx, &commands.PublishResourceLinksRequest{
		DeviceId: link.DeviceID,
		Resources: []*commands.Resource{{
			Href:                  href,
			ResourceTypes:         link.ResourceTypes,
			Interfaces:            link.Interfaces,
			DeviceId:              link.DeviceID,
			Anchor:                link.Anchor,
			Policy:                &commands.Policy{BitFlags: int32(link.Policy.BitMask)},
			Title:                 link.Title,
			SupportedContentTypes: link.SupportedContentTypes,
			EndpointInformations:  endpoints,
		}},
		CommandMetadata: cmdMetadata,
	})
	return err
}

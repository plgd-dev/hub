package service

import (
	"context"

	kitHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/sdk/schema"
)

func publishResource(ctx context.Context, raClient raService.ResourceAggregateClient, userID string, link schema.ResourceLink, cmdMetadata *commands.CommandMetadata) error {
	endpoints := make([]*commands.EndpointInformation, 0, 4)
	for _, endpoint := range link.GetEndpoints() {
		endpoints = append(endpoints, &commands.EndpointInformation{
			Endpoint: endpoint.URI,
			Priority: int64(endpoint.Priority),
		})
	}
	href := kitHttp.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
	_, err := raClient.PublishResourceLinks(ctx, &commands.PublishResourceLinksRequest{
		DeviceId: link.DeviceID,
		Resources: []*commands.Resource{{
			Href:                  href,
			ResourceTypes:         link.ResourceTypes,
			Interfaces:            link.Interfaces,
			DeviceId:              link.DeviceID,
			Anchor:                link.Anchor,
			Policies:              &commands.Policies{BitFlags: int32(link.Policy.BitMask)},
			Title:                 link.Title,
			SupportedContentTypes: link.SupportedContentTypes,
			EndpointInformations:  endpoints,
		}},
		CommandMetadata: cmdMetadata,
	})
	return err
}

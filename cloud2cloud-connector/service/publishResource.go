package service

import (
	"context"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	kitHttp "github.com/plgd-dev/kit/net/http"
	"github.com/plgd-dev/sdk/schema"
)

func publishResource(ctx context.Context, raClient pbRA.ResourceAggregateClient, userID string, link schema.ResourceLink, cmdMetadata pbCQRS.CommandMetadata) error {
	endpoints := make([]*pbRA.EndpointInformation, 0, 4)
	for _, endpoint := range link.GetEndpoints() {
		endpoints = append(endpoints, &pbRA.EndpointInformation{
			Endpoint: endpoint.URI,
			Priority: int64(endpoint.Priority),
		})
	}
	href := kitHttp.CanonicalHref(trimDeviceIDFromHref(link.DeviceID, link.Href))
	resourceID := utils.MakeResourceId(link.DeviceID, href)
	_, err := raClient.PublishResource(kitNetGrpc.CtxWithUserID(ctx, userID), &pbRA.PublishResourceRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			DeviceId: link.DeviceID,
		},
		ResourceId: &pbRA.ResourceId{
			DeviceId: link.DeviceID,
			Href:     href,
		},
		Resource: &pbRA.Resource{
			Id:                    resourceID,
			Href:                  href,
			ResourceTypes:         link.ResourceTypes,
			Interfaces:            link.Interfaces,
			DeviceId:              link.DeviceID,
			Anchor:                link.Anchor,
			Policies:              &pbRA.Policies{BitFlags: int32(link.Policy.BitMask)},
			Title:                 link.Title,
			SupportedContentTypes: link.SupportedContentTypes,
			EndpointInformations:  endpoints,
		},
		CommandMetadata: &cmdMetadata,
	})
	return err
}

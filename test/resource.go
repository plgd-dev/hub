package test

import (
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

func SortResources(s commands.Resources) commands.Resources {
	s.Sort()
	return s
}

func ResourceLinksToResources(deviceID string, s []schema.ResourceLink) []*commands.Resource {
	r := make([]*commands.Resource, 0, len(s))
	for _, l := range s {
		l.DeviceID = deviceID
		r = append(r, commands.SchemaResourceLinkToResource(l, time.Time{}))
	}
	CleanUpResourcesArray(r)
	return r
}

func ResourceLinksToResourceIds(deviceID string, s []schema.ResourceLink) []*commands.ResourceId {
	r := make([]*commands.ResourceId, 0, len(s))
	for _, l := range s {
		l.DeviceID = deviceID
		r = append(r, commands.SchemaResourceLinkToResourceId(l))
	}
	return r
}

func ResourceLinksToResources2(deviceID string, s []schema.ResourceLink) []*pb.Resource {
	r := make([]*pb.Resource, 0, len(s))
	for _, l := range s {
		res := pb.Resource{
			Types: l.ResourceTypes,
			Data: &events.ResourceChanged{
				ResourceId: &commands.ResourceId{
					DeviceId: deviceID,
					Href:     l.Href,
				},
			},
		}
		r = append(r, &res)
	}
	return r
}

func CleanUpResourcesArray(resources []*commands.Resource) []*commands.Resource {
	for _, r := range resources {
		r.ValidUntil = 0
	}
	SortResources(resources)
	return resources
}

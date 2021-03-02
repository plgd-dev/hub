package service

import (
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func toResourceValue(resource *Resource) pb.ResourceValue {
	return pb.ResourceValue{
		ResourceId: &commands.ResourceId{
			Href:     resource.Resource.GetHref(),
			DeviceId: resource.Resource.GetDeviceId(),
		},
		Content: pb.RAContent2Content(resource.GetContent()),
		Types:   resource.Resource.GetResourceTypes(),
		Status:  pb.RAStatus2Status(resource.GetStatus()),
	}
}

type ResourceShadow struct {
	projection    *Projection
	userDeviceIds strings.Set
}

func NewResourceShadow(projection *Projection, deviceIds []string) *ResourceShadow {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceShadow{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceShadow) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)

	resourceIDsFilter := make([]*commands.ResourceId, 0, len(req.GetResourceIdsFilter())+len(req.GetDeviceIdsFilter()))
	for _, res := range req.GetResourceIdsFilter() {
		if rd.userDeviceIds.HasOneOf(res.GetDeviceId()) {
			resourceIDsFilter = append(resourceIDsFilter, res)
		}
	}
	for _, deviceID := range req.GetDeviceIdsFilter() {
		if rd.userDeviceIds.HasOneOf(deviceID) {
			resourceIDsFilter = append(resourceIDsFilter, commands.NewResourceID(deviceID, ""))
		}
	}
	if len(resourceIDsFilter) == 0 {
		if len(req.GetResourceIdsFilter()) > 0 || len(req.GetDeviceIdsFilter()) > 0 {
			return status.Errorf(codes.NotFound, "resource ids filter doesn't match any resources")
		}
		resourceIDsFilter = make([]*commands.ResourceId, 0, len(rd.userDeviceIds))
		for userDeviceID := range rd.userDeviceIds {
			resourceIDsFilter = append(resourceIDsFilter, commands.NewResourceID(userDeviceID, ""))
		}
	}

	resources, err := rd.projection.GetResourcesWithLinks(srv.Context(), resourceIDsFilter, typeFilter)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot retrieve resources: %v", err)
	}
	if len(resources) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, deviceResources := range resources {
		for _, resource := range deviceResources {
			val := toResourceValue(resource)
			err = srv.Send(&val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot retrieve resources projections: %v", err)
			}
		}
	}
	return nil
}

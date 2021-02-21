package service

import (
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func toResourceValue(m *resourceProjection, resource *commands.Resource) pb.ResourceValue {
	return pb.ResourceValue{
		ResourceId: &commands.ResourceId{
			Href:     m.resourceId.GetHref(),
			DeviceId: m.resourceId.GetDeviceId(),
		},
		Content: pb.RAContent2Content(m.content.GetContent()),
		Types:   resource.GetResourceTypes(),
		Status:  pb.RAStatus2Status(m.content.GetStatus()),
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
	deviceIDs := filterDevices(rd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIDs) == 0 {
		return status.Errorf(codes.NotFound, "device ids filter doesn't match any devices")
	}

	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)

	// validate access to resource
	resourceIDsFilter := make([]*commands.ResourceId, 0, 64)
	for _, res := range req.GetResourceIdsFilter() {
		if len(filterDevices(rd.userDeviceIds, []string{res.GetDeviceId()})) > 0 {
			resourceIDsFilter = append(resourceIDsFilter, res)
		}
	}
	if len(resourceIDsFilter) == 0 && len(req.GetResourceIdsFilter()) > 0 && len(req.GetDeviceIdsFilter()) == 0 {
		return status.Errorf(codes.NotFound, "resource ids filter doesn't match any resources")
	}
	if len(req.GetResourceIdsFilter()) > 0 && len(req.GetDeviceIdsFilter()) == 0 {
		deviceIDs = strings.MakeSet()
	}

	resourceProjections, err := rd.projection.GetResourceProjections(srv.Context(), resourceIDsFilter, typeFilter)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot retrieve resources projections: %v", err)
	}
	if len(resourceProjections) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	resourceLinks, err := rd.projection.GetResourceLinks(srv.Context(), deviceIDs, typeFilter)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot retrieve resources links: %v", err)
	}

	for _, resources := range resourceProjections {
		for _, resourceProjection := range resources {
			resource := resourceLinks[resourceProjection.resourceId.GetDeviceId()][resourceProjection.resourceId.GetHref()]
			val := toResourceValue(resourceProjection, resource)
			err = srv.Send(&val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot retrieve resources projections: %v", err)
			}
		}
	}
	return nil
}

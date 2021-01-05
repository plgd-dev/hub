package service

import (
	"github.com/plgd-dev/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

func toResourceValue(m *resourceCtx) pb.ResourceValue {
	return pb.ResourceValue{
		ResourceId: &pb.ResourceId{
			Href:     m.resource.GetHref(),
			DeviceId: m.resource.GetDeviceId(),
		},
		Content: pb.RAContent2Content(m.content.GetContent()),
		Types:   m.resource.GetResourceTypes(),
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
	deviceIds := filterDevices(rd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		return status.Errorf(codes.NotFound, "device ids filter doesn't match any devices")
	}
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)

	// validate access to resource
	resourceIdsFilter := make([]*pb.ResourceId, 0, 64)
	for _, res := range req.GetResourceIdsFilter() {
		if len(filterDevices(rd.userDeviceIds, []string{res.GetDeviceId()})) > 0 {
			resourceIdsFilter = append(resourceIdsFilter, res)
		}
	}
	if len(resourceIdsFilter) == 0 && len(req.GetResourceIdsFilter()) > 0 && len(req.GetDeviceIdsFilter()) == 0 {
		return status.Errorf(codes.NotFound, "resource ids filter doesn't match any resources")
	}
	if len(req.GetResourceIdsFilter()) > 0 && len(req.GetDeviceIdsFilter()) == 0 {
		deviceIds = strings.MakeSet()
	}

	resourceValues, err := rd.projection.GetResourceCtxs(srv.Context(), resourceIdsFilter, typeFilter, deviceIds)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot retrieve resources values: %v", err)
	}
	if len(resourceValues) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, resources := range resourceValues {
		for _, resource := range resources {
			val := toResourceValue(resource)
			err = srv.Send(&val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot retrieve resources values: %v", err)
			}
		}
	}
	return nil
}

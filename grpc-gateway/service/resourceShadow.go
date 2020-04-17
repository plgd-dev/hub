package service

import (
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

func toResourceValue(m *resourceCtx) pb.ResourceValue {
	return pb.ResourceValue{
		ResourceId: &pb.ResourceId{
			ResourceLinkHref: m.resource.GetHref(),
			DeviceId:         m.resource.GetDeviceId(),
		},
		Content: pb.RAContent2Content(m.content.GetContent()),
		Types:   m.resource.GetResourceTypes(),
		Status:  pb.RAStatus2Status(m.content.Status),
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
		return status.Errorf(codes.NotFound, "not found")
	}
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	resourceIdsFilter := make(strings.Set)
	for _, r := range req.ResourceIdsFilter {
		resourceIdsFilter.Add(r.ID())
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
			if resource.content == nil {
				continue
			}
			val := toResourceValue(resource)
			err = srv.Send(&val)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot retrieve resources values: %v", err)
			}
		}
	}
	return nil
}

package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/strings"
)

type ResourceDirectory struct {
	projection    *Projection
	userDeviceIds strings.Set
}

func New(projection *Projection, deviceIds []string) *ResourceDirectory {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceDirectory{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceDirectory) GetResourceLinks(in *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	deviceIds := filterDevices(rd.userDeviceIds, in.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	typeFilter := make(strings.Set)
	typeFilter.Add(in.TypeFilter...)

	resourceValues, err := rd.projection.GetResourceCtxs(srv.Context(), nil, typeFilter, deviceIds)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resource links by device ids: %v", err)
	}
	if len(resourceValues) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, resources := range resourceValues {
		for _, resource := range resources {
			if resource.resourceId == nil {
				continue
			}
			resourceLink := pb.RAResourceToProto(resource.resourceId)
			err = srv.Send(&resourceLink)
			if err != nil {
				return status.Errorf(codes.Canceled, "cannot send resource link: %v", err)
			}
		}
	}
	return nil
}

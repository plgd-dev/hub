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

func NewResourceDirectory(projection *Projection, deviceIds []string) *ResourceDirectory {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceDirectory{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceDirectory) GetResourceLinks(in *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	deviceIDs := filterDevices(rd.userDeviceIds, in.DeviceIdsFilter)
	if len(deviceIDs) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	typeFilter := make(strings.Set)
	typeFilter.Add(in.TypeFilter...)

	resourceLinks, err := rd.projection.GetResourceLinks(srv.Context(), deviceIDs, typeFilter)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get resource links %v", err)
	}
	if len(resourceLinks) == 0 {
		return status.Errorf(codes.NotFound, "not found")
	}

	for _, s := range resourceLinks {
		err = srv.Send(s.ToResourceLinksPublished())
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send resource link %v", err)
		}
	}
	return nil
}

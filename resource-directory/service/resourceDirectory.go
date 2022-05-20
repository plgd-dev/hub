package service

import (
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (rd *ResourceDirectory) sendResourceLinks(srv pb.GrpcGateway_GetResourceLinksServer, deviceIDs, typeFilter, toReloadDevices strings.Set) error {
	return rd.projection.LoadResourceLinks(srv.Context(), deviceIDs, toReloadDevices, func(m *resourceLinksProjection) error {
		p := m.Clone()
		for href, resource := range p.snapshot.GetResources() {
			if !hasMatchingType(resource.ResourceTypes, typeFilter) {
				delete(p.snapshot.Resources, href)
			}
		}
		if len(p.snapshot.Resources) == 0 {
			return nil
		}
		err := srv.Send(p.snapshot.ToResourceLinksPublished())
		if err != nil {
			return status.Errorf(codes.Canceled, "cannot send resource link %v", err)
		}
		return nil
	})
}

func (rd *ResourceDirectory) GetResourceLinks(in *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	deviceIDs := filterDevices(rd.userDeviceIds, in.DeviceIdFilter)
	if len(deviceIDs) == 0 {
		log.Debug("ResourceDirectory.GetResourceLinks.filterDevices returns empty deviceIDs")
		return nil
	}

	typeFilter := make(strings.Set)
	typeFilter.Add(in.TypeFilter...)

	toReloadDevices := make(strings.Set)
	err := rd.sendResourceLinks(srv, deviceIDs, typeFilter, toReloadDevices)
	if err != nil {
		return err
	}
	if len(toReloadDevices) > 0 {
		rd.projection.ReloadDevices(srv.Context(), toReloadDevices)
		return rd.sendResourceLinks(srv, toReloadDevices, typeFilter, nil)
	}

	return nil
}

package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/strings"
)

func toResourceLink(model *resourceCtx) pbRD.ResourceLink {
	return pbRD.ResourceLink{Resource: model.snapshot.Resource}
}

type ResourceDirectory struct {
	projection    *Projection
	userDeviceIds strings.Set
}

func NewResourceDirectory(projection *Projection, deviceIds []string) *ResourceDirectory {
	mapDeviceIds := make(strings.Set)
	mapDeviceIds.Add(deviceIds...)

	return &ResourceDirectory{projection: projection, userDeviceIds: mapDeviceIds}
}

func (rd *ResourceDirectory) GetResourceLinks(ctx context.Context, in *pbRD.GetResourceLinksRequest, responseHandler func(*pbRD.ResourceLink) error) (statusCode codes.Code, err error) {
	deviceIds := filterDevices(rd.userDeviceIds, in.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}

	typeFilter := make(strings.Set)
	typeFilter.Add(in.TypeFilter...)
	resourceIdsFilter := make(strings.Set)

	resourceValues, err := rd.projection.GetResourceCtxs(ctx, resourceIdsFilter, typeFilter, deviceIds)
	if err != nil {
		err = fmt.Errorf("cannot get resource links by device ids: %w", err)
		statusCode = codes.Internal
		return
	}
	if len(resourceValues) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}

	links := make([]pbRD.ResourceLink, 0, 32)
	for _, resources := range resourceValues {
		for _, resource := range resources {
			resourceLink := toResourceLink(resource)
			links = append(links, resourceLink)
			if err = responseHandler(&resourceLink); err != nil {
				err = fmt.Errorf("cannot handle response: %w", err)
				statusCode = codes.Canceled
				return
			}
		}
	}
	log.Debugf("DeviceDirectory.GetResourceLinks send links %+v", links)
	statusCode = codes.OK
	return
}

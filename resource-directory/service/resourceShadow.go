package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/strings"
	"google.golang.org/grpc/codes"

	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
)

func toResourceValue(m *resourceCtx) pbRS.ResourceValue {
	return pbRS.ResourceValue{
		ResourceId: m.snapshot.GetResource().GetId(),
		DeviceId:   m.snapshot.GetResource().GetDeviceId(),
		Href:       m.snapshot.GetResource().GetHref(),
		Content:    m.snapshot.GetLatestResourceChange().GetContent(),
		Types:      m.snapshot.GetResource().GetResourceTypes(),
		Status:     m.snapshot.GetLatestResourceChange().GetStatus(),
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

// filterDevices returns filtered device ids that match filter.
// An empty deviceIdsFilter matches all device ids.
func filterDevices(deviceIds strings.Set, deviceIdsFilter []string) strings.Set {
	if len(deviceIdsFilter) == 0 {
		return deviceIds
	}
	result := make(strings.Set)
	for _, deviceId := range deviceIdsFilter {
		if deviceIds.HasOneOf(deviceId) {
			result.Add(deviceId)
		}
	}
	return result
}

func (rd *ResourceShadow) RetrieveResourcesValues(ctx context.Context, req *pbRS.RetrieveResourcesValuesRequest, responseHandler func(*pbRS.ResourceValue) error) (statusCode codes.Code, err error) {
	deviceIds := filterDevices(rd.userDeviceIds, req.DeviceIdsFilter)
	if len(deviceIds) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}
	typeFilter := make(strings.Set)
	typeFilter.Add(req.TypeFilter...)
	resourceIdsFilter := make(strings.Set)
	resourceIdsFilter.Add(req.ResourceIdsFilter...)

	resourceValues, err := rd.projection.GetResourceCtxs(ctx, resourceIdsFilter, typeFilter, deviceIds)
	if err != nil {
		err = fmt.Errorf("cannot retrieve resources values: %w", err)
		statusCode = codes.Internal
		return
	}
	if len(resourceValues) == 0 {
		err = fmt.Errorf("not found")
		statusCode = codes.NotFound
		return
	}

	res := make([]pbRS.ResourceValue, 0, 32)
	for _, resources := range resourceValues {
		for _, resource := range resources {
			val := toResourceValue(resource)
			res = append(res, val)
			if err = responseHandler(&val); err != nil {
				err = fmt.Errorf("cannot retrieve resources values: %w", err)
				statusCode = codes.Canceled
				return
			}
		}
	}
	log.Debugf("DeviceDirectory.RetrieveResourcesValues send resources %+v", res)
	statusCode = codes.OK
	return
}

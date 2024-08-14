package client

import (
	"context"
	"net/http"
	"slices"

	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reboot makes reboot on device. JWT token must be stored in context for grpc call.
func (c *Client) Reboot(
	ctx context.Context,
	deviceID string,
) error {
	return c.updateMaintenanceResource(ctx, deviceID, maintenance.MaintenanceUpdateRequest{
		Reboot: true,
	})
}

// FactoryReset makes factory reset on device. JWT token must be stored in context for grpc call.
func (c *Client) FactoryReset(
	ctx context.Context,
	deviceID string,
) error {
	return c.updateMaintenanceResource(ctx, deviceID, maintenance.MaintenanceUpdateRequest{
		FactoryReset: true,
	})
}

// https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
func httpCoreToGrpc(statusCode int) codes.Code {
	switch statusCode {
	case http.StatusBadRequest:
		return codes.Internal
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.Unimplemented
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return codes.Unavailable
	}
	return codes.Unknown
}

func (c *Client) updateMaintenanceResource(
	ctx context.Context,
	deviceID string,
	req maintenance.MaintenanceUpdateRequest,
) (ret error) {
	it := c.GetResourceLinksIterator(ctx, []string{deviceID}, maintenance.ResourceType)
	defer it.Close()
	var v events.ResourceLinksPublished
	if !it.Next(&v) {
		if it.Err != nil {
			return grpc.ForwardErrorf(codes.NotFound, "cannot find maintenance resource(%v): %v", maintenance.ResourceType, it.Err)
		}
		return status.Errorf(codes.NotFound, "cannot find maintenance resource(%v)", maintenance.ResourceType)
	}
	var href string
	for _, r := range v.GetResources() {
		if r.GetDeviceId() == deviceID && slices.Contains(r.GetResourceTypes(), maintenance.ResourceType) {
			href = r.GetHref()
			break
		}
	}

	var resp maintenance.Maintenance
	err := c.UpdateResource(ctx, v.GetDeviceId(), href, req, &resp)
	if err != nil {
		return err
	}
	if resp.LastHTTPError >= http.StatusBadRequest {
		defer func() {
			if r := recover(); r != nil {
				ret = status.Errorf(httpCoreToGrpc(resp.LastHTTPError), "returns HTTP code %v", resp.LastHTTPError)
			}
		}()
		str := http.StatusText(resp.LastHTTPError)
		return status.Errorf(httpCoreToGrpc(resp.LastHTTPError), "%s", str)
	}
	return it.Err
}

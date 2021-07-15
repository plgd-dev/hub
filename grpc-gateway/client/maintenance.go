package client

import (
	"context"
	"net/http"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/sdk/schema/maintenance"
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

//https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
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
	it := c.GetResourceLinksIterator(ctx, []string{deviceID}, maintenance.MaintenanceResourceType)
	defer it.Close()
	var v commands.Resource
	if !it.Next(&v) {
		return status.Errorf(codes.NotFound, "cannot find maintenance resource(%v)", maintenance.MaintenanceResourceType)
	}
	var resp maintenance.Maintenance
	err := c.UpdateResource(ctx, v.GetDeviceId(), v.GetHref(), req, &resp)
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
		return status.Errorf(httpCoreToGrpc(resp.LastHTTPError), str)
	}
	return it.Err
}

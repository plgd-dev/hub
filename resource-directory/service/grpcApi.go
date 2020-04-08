package service

import (
	"context"
	"fmt"
	"io"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	authClient pbAS.AuthorizationServiceClient
	projection *Projection
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(authClient pbAS.AuthorizationServiceClient, projection *Projection) *RequestHandler {
	return &RequestHandler{
		authClient: authClient,
		projection: projection,
	}
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}

func (r *RequestHandler) GetUsersDevices(ctx context.Context, authCtx *pbCQRS.AuthorizationContext, deviceIdsFilter []string) ([]string, error) {
	userIdsFilter := []string(nil)
	if authCtx.GetUserId() != "" {
		userIdsFilter = []string{authCtx.GetUserId()}
	}
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "cannot get users devices: %v", err)
	}
	getUserDevicesClient, err := r.authClient.GetUserDevices(kitNetGrpc.CtxWithToken(ctx, token), &pbAS.GetUserDevicesRequest{
		UserIdsFilter:   userIdsFilter,
		DeviceIdsFilter: deviceIdsFilter,
	})
	if err != nil {
		return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
	}
	userDevices := make([]string, 0, 32)
	for {
		userDevice, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
		}
		if userDevice == nil {
			continue
		}
		userDevices = append(userDevices, userDevice.DeviceId)
	}
	return userDevices, nil
}

func (r *RequestHandler) RetrieveResourcesValues(req *pbRS.RetrieveResourcesValuesRequest, srv pbRS.ResourceShadow_RetrieveResourcesValuesServer) error {
	deviceIds, err := r.GetUsersDevices(srv.Context(), req.GetAuthorizationContext(), req.DeviceIdsFilter)
	if err != nil {
		return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot retrieve resources values: %v", err))
	}
	if len(deviceIds) == 0 {
		return logAndReturnError(status.Errorf(codes.NotFound, "cannot retrieve resources values: not found"))
	}

	rd := NewResourceShadow(r.projection, deviceIds)

	statusCode, err := rd.RetrieveResourcesValues(srv.Context(), req, func(resourceLink *pbRS.ResourceValue) error {
		err := srv.Send(resourceLink)
		if err != nil {
			return fmt.Errorf("cannot send resource value to client: %v", err)
		}
		return nil
	})
	if err != nil {
		return logAndReturnError(status.Errorf(statusCode, "cannot retrieve resources values: %v", err))
	}
	return nil
}

func (r *RequestHandler) GetResourceLinks(req *pbRD.GetResourceLinksRequest, srv pbRD.ResourceDirectory_GetResourceLinksServer) error {
	deviceIds, err := r.GetUsersDevices(srv.Context(), req.GetAuthorizationContext(), req.DeviceIdsFilter)
	if err != nil {
		return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot retrieve resources values: %v", err))
	}
	if len(deviceIds) == 0 {
		return logAndReturnError(status.Errorf(codes.NotFound, "cannot retrieve resources values: not found"))
	}

	rd := NewResourceDirectory(r.projection, deviceIds)

	code, err := rd.GetResourceLinks(srv.Context(), req, func(resourceLink *pbRD.ResourceLink) error {
		err := srv.Send(resourceLink)
		if err != nil {
			return fmt.Errorf("cannot send resource link to client: %v", err)
		}
		return nil
	})

	if err != nil {
		return logAndReturnError(status.Errorf(code, "cannot get resource links: %v", err))
	}
	return nil
}

func (r *RequestHandler) GetDevices(req *pbDD.GetDevicesRequest, srv pbDD.DeviceDirectory_GetDevicesServer) error {
	deviceIds, err := r.GetUsersDevices(srv.Context(), req.GetAuthorizationContext(), req.DeviceIdsFilter)
	if err != nil {
		return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get devices contents: %v", err))
	}

	rd := NewDeviceDirectory(r.projection, deviceIds)

	code, err := rd.GetDevices(srv.Context(), req, func(device *pbDD.Device) error {
		err := srv.Send(device)
		if err != nil {
			return fmt.Errorf("cannot send device to client: %v", err)
		}
		return nil
	})
	if err != nil {
		return logAndReturnError(status.Errorf(code, "cannot get devices contents: %v", err))
	}
	return err
}

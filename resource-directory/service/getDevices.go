package service

import (
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	owner, err := kitNetGrpc.OwnerFromMD(srv.Context())
	if err != nil {
		return log.LogAndReturnError(status.Errorf(codes.Unauthenticated, "cannot get devices: %v", err))
	}
	deviceIDs, err := r.GetOwnerDevices(srv.Context(), owner)
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get devices contents: %v", err))
	}

	rd := NewDeviceDirectory(r.resourceProjection, deviceIDs)
	err = rd.GetDevices(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}

	return nil
}

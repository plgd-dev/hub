package service

import (
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetDevicesMetadata(req *pb.GetDevicesMetadataRequest, srv pb.GrpcGateway_GetDevicesMetadataServer) error {
	owner, err := kitNetGrpc.OwnerFromTokenMD(srv.Context(), r.ownerCache.OwnerClaim())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.getOwnerDevices(srv.Context(), owner)
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot retrieve devices metadata: %v", err))
	}

	rs := NewResourceShadow(r.resourceProjection, deviceIDs)
	err = rs.GetDevicesMetadata(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}

	return nil

}

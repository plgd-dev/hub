package service

import (
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	owner, err := kitNetGrpc.OwnerFromTokenMD(srv.Context(), r.ownerCache.OwnerClaim())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.getOwnerDevices(srv.Context(), owner)
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get devices contents: %v", err))
	}

	rs := NewResourceShadow(r.resourceProjection, deviceIDs)
	err = rs.GetResources(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}
	return nil
}

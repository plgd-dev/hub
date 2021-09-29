package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) getOwnerDevices(ctx context.Context, owner string) ([]string, error) {
	deviceIDs, err := r.ownerCache.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	return deviceIDs, err
}

func (r *RequestHandler) GetPendingCommands(req *pb.GetPendingCommandsRequest, srv pb.GrpcGateway_GetPendingCommandsServer) error {
	owner, err := kitNetGrpc.OwnerFromTokenMD(srv.Context(), r.ownerCache.OwnerClaim())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.getOwnerDevices(srv.Context(), owner)
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot retrieve pending commands: %v", err))
	}

	rs := NewResourceShadow(r.resourceProjection, deviceIDs)
	err = rs.GetPendingCommands(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}

	return nil
}

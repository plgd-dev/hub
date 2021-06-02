package service

import (
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) RetrieveDevicesMetadata(req *pb.RetrieveDevicesMetadataRequest, srv pb.GrpcGateway_RetrieveDevicesMetadataServer) error {
	owner, err := kitNetGrpc.OwnerFromMD(srv.Context())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.userDevicesManager.GetUserDevices(srv.Context(), owner)
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot retrieve devices metadata: %v", err))
	}

	rs := NewResourceShadow(r.resourceProjection, deviceIDs)
	err = rs.RetrieveDevicesMetadata(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}

	return nil

}

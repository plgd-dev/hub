package service

import (
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	userID, err := kitNetGrpc.UserIDFromMD(srv.Context())
	if err != nil {
		return kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.userDevicesManager.GetUserDevices(srv.Context(), userID)
	if err != nil {
		return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get devices contents: %v", err))
	}

	rs := NewResourceShadow(r.resourceProjection, deviceIDs)
	err = rs.RetrieveResourcesValues(req, srv)
	if err != nil {
		return logAndReturnError(err)
	}
	return nil
}

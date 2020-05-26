package service

import (
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	accessToken, err := grpc_auth.AuthFromMD(srv.Context(), "bearer")
	if err != nil {
		return logAndReturnError(status.Errorf(codes.Unauthenticated, "cannot retrieve resources values: %v", err))
	}
	userID, err := parseSubFromJwtToken(accessToken)
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

package service

import (
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	userID, err := kitNetGrpc.UserIDFromMD(srv.Context())
	if err != nil {
		return logAndReturnError(kitNetGrpc.ForwardErrorf(codes.NotFound, "cannot get resource links: %v", err))
	}
	deviceIDs, err := r.userDevicesManager.GetUserDevices(srv.Context(), userID)
	if err != nil {
		return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get devices contents: %v", err))
	}

	rd := NewResourceDirectory(r.resourceProjection, deviceIDs)
	err = rd.GetResourceLinks(req, srv)
	if err != nil {
		return logAndReturnError(err)
	}
	return nil
}

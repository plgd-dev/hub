package service

import (
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	_, err := kitNetGrpc.OwnerFromTokenMD(srv.Context(), r.ownerCache.OwnerClaim())
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Unauthenticated, "cannot get resource links: %v", err))
	}
	deviceIDs, err := r.getOwnerDevices(srv.Context())
	if err != nil {
		return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get resource links: %v", err))
	}

	rd := NewResourceDirectory(r.resourceProjection, deviceIDs)
	err = rd.GetResourceLinks(req, srv)
	if err != nil {
		return log.LogAndReturnError(err)
	}
	return nil
}

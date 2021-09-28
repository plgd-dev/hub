package service

import (
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	owner, err := kitNetGrpc.OwnerFromMD(srv.Context())
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Unauthenticated, "cannot get resource links: %v", err))
	}
	deviceIDs, err := r.GetOwnerDevices(srv.Context(), owner)
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

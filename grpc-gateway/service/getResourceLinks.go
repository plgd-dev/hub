package service

import (
	"errors"
	"io"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.GetResourceLinks(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get resource links: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive link: %v", err)
		}
		err = srv.Send(resp)
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send link: %v", err)
		}
	}
	return nil
}

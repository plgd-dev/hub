package service

import (
	"io"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	ctx := makeCtx(srv.Context())
	rd, err := r.resourceDirectoryClient.GetResourceLinks(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get resource links: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if err == io.EOF {
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

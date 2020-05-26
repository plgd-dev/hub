package service

import (
	"io"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	ctx := makeCtx(srv.Context())
	rd, err := r.resourceDirectoryClient.RetrieveResourcesValues(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resources values: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive resource: %v", err)
		}
		err = srv.Send(resp)
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send resource: %v", err)
		}
	}
	return nil
}

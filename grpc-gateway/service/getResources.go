package service

import (
	"errors"
	"io"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.GetResources(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resources values: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if errors.Is(err, io.EOF) {
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

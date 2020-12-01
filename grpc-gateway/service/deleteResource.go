package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	ret, err := r.resourceDirectoryClient.DeleteResource(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err)
	}
	return ret, err
}

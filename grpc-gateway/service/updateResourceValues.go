package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) UpdateResourcesValues(ctx context.Context, req *pb.UpdateResourceValuesRequest) (*pb.UpdateResourceValuesResponse, error) {
	ret, err := r.resourceDirectoryClient.UpdateResourcesValues(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resources values: %v", err)
	}
	return ret, err
}

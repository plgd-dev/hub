package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) UpdateResourcesValues(ctx context.Context, req *pb.UpdateResourceValuesRequest) (*pb.UpdateResourceValuesResponse, error) {
	ret, err := r.resourceDirectoryClient.UpdateResourcesValues(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resources values: %v", err)
	}
	return ret, err
}

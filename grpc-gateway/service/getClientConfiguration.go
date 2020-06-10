package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetClientConfiguration(ctx context.Context, req *pb.ClientConfigurationRequest) (*pb.ClientConfigurationResponse, error) {
	ret, err := r.resourceDirectoryClient.GetClientConfiguration(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get client configuration: %v", err)
	}
	return ret, err
}

package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetHubConfiguration(ctx context.Context, req *pb.HubConfigurationRequest) (*pb.HubConfigurationResponse, error) {
	ret, err := r.resourceDirectoryClient.GetHubConfiguration(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get client configuration: %v", err)
	}
	return ret, err
}

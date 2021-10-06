package service

import (
	"context"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetCloudConfiguration(ctx context.Context, req *pb.CloudConfigurationRequest) (*pb.CloudConfigurationResponse, error) {
	ret, err := r.resourceDirectoryClient.GetCloudConfiguration(ctx, req)
	if err != nil {
		return ret, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get client configuration: %v", err))
	}
	return ret, err
}

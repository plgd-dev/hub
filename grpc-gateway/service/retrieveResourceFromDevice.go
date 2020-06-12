package service

import (
	"context"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) RetrieveResourceFromDevice(ctx context.Context, req *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	ret, err := r.resourceDirectoryClient.RetrieveResourceFromDevice(ctx, req)
	if err != nil {
		return ret, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource from device: %v", err)
	}
	return ret, err
}

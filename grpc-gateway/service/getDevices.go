package service

import (
	"io"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	ctx := makeCtx(srv.Context())
	rd, err := r.resourceDirectoryClient.GetDevices(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get devices: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive device: %v", err)
		}
		err = srv.Send(resp)
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send device: %v", err)
		}
	}
	return nil
}

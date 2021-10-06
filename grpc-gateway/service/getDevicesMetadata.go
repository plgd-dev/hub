package service

import (
	"io"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetDevicesMetadata(req *pb.GetDevicesMetadataRequest, srv pb.GrpcGateway_GetDevicesMetadataServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.GetDevicesMetadata(ctx, req)
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve devices metadata: %v", err))
	}
	for {
		resp, err := rd.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive devices metadata: %v", err))
		}
		err = srv.Send(resp)
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send devices metadata('%v'): %v", resp, err))
		}
	}
	return nil

}

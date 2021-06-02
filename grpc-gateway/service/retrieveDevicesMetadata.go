package service

import (
	"io"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) RetrieveDevicesMetadata(req *pb.RetrieveDevicesMetadataRequest, srv pb.GrpcGateway_RetrieveDevicesMetadataServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.RetrieveDevicesMetadata(ctx, req)
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
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send devices metadata: %v", err))
		}
	}
	return nil

}

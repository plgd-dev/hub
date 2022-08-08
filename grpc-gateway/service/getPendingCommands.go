package service

import (
	"errors"
	"io"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetPendingCommands(req *pb.GetPendingCommandsRequest, srv pb.GrpcGateway_GetPendingCommandsServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.GetPendingCommands(ctx, req)
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve pending commands: %v", err))
	}
	for {
		resp, err := rd.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive pending command: %v", err))
		}
		err = srv.Send(resp)
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send pending command: %v", err))
		}
	}
	return nil
}

package service

import (
	"errors"
	"io"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) GetEvents(req *pb.GetEventsRequest, srv pb.GrpcGateway_GetEventsServer) error {
	ctx := srv.Context()
	rd, err := r.resourceDirectoryClient.GetEvents(ctx, req)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot get events: %v", err)
	}
	for {
		resp, err := rd.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive event: %v", err)
		}
		err = srv.Send(resp)
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send event: %v", err)
		}
	}
	return nil
}

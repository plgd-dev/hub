package service

import (
	"context"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
)

func (r *RequestHandler) CancelPendingCommands(ctx context.Context, req *pb.CancelPendingCommandsRequest) (*pb.CancelPendingCommandsResponse, error) {
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	resp, err := r.resourceAggregateClient.CancelPendingCommands(ctx, &commands.CancelPendingCommandsRequest{
		ResourceId:          req.GetResourceId(),
		CorrelationIdFilter: req.GetCorrelationIdFilter(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel resource('%v') commands: %v", req.GetResourceId().ToString(), err))
	}

	return &pb.CancelPendingCommandsResponse{
		CorrelationIds: resp.GetCorrelationIds(),
	}, nil
}

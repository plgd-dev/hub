package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
)

func (r *RequestHandler) CancelDeviceMetadataUpdates(ctx context.Context, req *pb.CancelDeviceMetadataUpdatesRequest) (*pb.CancelResponse, error) {
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	resp, err := r.resourceAggregateClient.CancelDeviceMetadataUpdates(ctx, &commands.CancelDeviceMetadataUpdatesRequest{
		DeviceId:            req.GetDeviceId(),
		CorrelationIdFilter: req.GetCorrelationIdFilter(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot cancel device('%v') metadata updates: %v", req.GetDeviceId(), err))
	}

	return &pb.CancelResponse{
		CorrelationIds: resp.GetCorrelationIds(),
	}, nil
}

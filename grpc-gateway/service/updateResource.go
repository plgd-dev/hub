package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func logAndReturnUpdateResourceError(err error) error {
	return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err))
}

func (r *RequestHandler) UpdateResource(ctx context.Context, req *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	updateCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, logAndReturnUpdateResourceError(err)
	}
	updatedEvent, err := r.resourceAggregateClient.SyncUpdateResource(ctx, "*", updateCommand)
	if err != nil {
		return nil, logAndReturnUpdateResourceError(err)
	}
	err = commands.CheckEventContent(updatedEvent)
	if err != nil {
		return nil, logAndReturnUpdateResourceError(err)
	}
	return &pb.UpdateResourceResponse{Data: updatedEvent}, nil
}

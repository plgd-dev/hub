package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func updateResourceError(err error) error {
	return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err)
}

func (r *RequestHandler) UpdateResource(ctx context.Context, req *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	updateCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, updateResourceError(err)
	}
	updatedEvent, err := r.resourceAggregateClient.SyncUpdateResource(ctx, "*", updateCommand)
	if err != nil {
		return nil, updateResourceError(err)
	}
	err = commands.CheckEventContent(updatedEvent)
	if err != nil {
		return nil, updateResourceError(err)
	}
	return &pb.UpdateResourceResponse{Data: updatedEvent}, nil
}

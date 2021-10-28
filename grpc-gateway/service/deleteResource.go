package service

import (
	"context"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func logAndReturnDeleteResourceError(err error) error {
	return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err))
}

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	deleteCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, logAndReturnDeleteResourceError(err)
	}
	deletedEvent, err := r.resourceAggregateClient.SyncDeleteResource(ctx, "*", deleteCommand)
	if err != nil {
		return nil, logAndReturnDeleteResourceError(err)
	}
	if err = commands.CheckEventContent(deletedEvent); err != nil {
		return nil, logAndReturnDeleteResourceError(err)
	}
	return &pb.DeleteResourceResponse{Data: deletedEvent}, err
}

package service

import (
	"context"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func logAndReturnCreateResourceError(err error) error {
	return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
}

func (r *RequestHandler) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*pb.CreateResourceResponse, error) {
	createCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, logAndReturnCreateResourceError(err)
	}

	createdEvent, err := r.resourceAggregateClient.SyncCreateResource(ctx, "*", createCommand)
	if err != nil {
		return nil, logAndReturnCreateResourceError(err)
	}
	if err = commands.CheckEventContent(createdEvent); err != nil {
		return nil, logAndReturnCreateResourceError(err)
	}
	return &pb.CreateResourceResponse{Data: createdEvent}, nil
}

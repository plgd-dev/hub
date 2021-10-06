package service

import (
	"context"

	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/v2/pkg/net/grpc"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*pb.CreateResourceResponse, error) {
	createCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}

	createdEvent, err := r.resourceAggregateClient.SyncCreateResource(ctx, "*", createCommand)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}
	err = commands.CheckEventContent(createdEvent)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}
	return &pb.CreateResourceResponse{Data: createdEvent}, nil
}

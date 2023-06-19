package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func createResourceError(err error) error {
	return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err)
}

func (r *RequestHandler) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*pb.CreateResourceResponse, error) {
	createCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, createResourceError(err)
	}

	createdEvent, err := r.resourceAggregateClient.SyncCreateResource(ctx, "*", createCommand)
	if err != nil {
		return nil, createResourceError(err)
	}
	if err = commands.CheckEventContent(createdEvent); err != nil {
		return nil, createResourceError(err)
	}
	return &pb.CreateResourceResponse{Data: createdEvent}, nil
}

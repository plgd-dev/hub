package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*events.ResourceCreated, error) {
	createCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}

	createdEvent, err := r.resourceAggregateClient.SyncCreateResource(ctx, createCommand)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot create resource: %v", err))
	}
	return createdEvent, nil
}

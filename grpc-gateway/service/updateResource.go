package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) UpdateResource(ctx context.Context, req *pb.UpdateResourceRequest) (*events.ResourceUpdated, error) {
	updateCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err))
	}
	updatedEvent, err := r.resourceAggregateClient.SyncUpdateResource(ctx, updateCommand)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err))
	}
	return updatedEvent, nil
}

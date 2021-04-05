package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) UpdateResource(ctx context.Context, req *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	updateCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err)
	}
	updatedEvent, err := r.resourceAggregateClient.SyncUpdateResource(ctx, updateCommand)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update resource: %v", err)
	}
	return pb.RAResourceUpdatedEventToResponse(updatedEvent)
}

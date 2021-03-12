package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	deleteCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err)
	}
	deletedEvent, err := r.resourceAggregateClient.SyncDeleteResource(ctx, deleteCommand)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err)
	}
	return pb.RAResourceDeletedEventToResponse(deletedEvent)
}

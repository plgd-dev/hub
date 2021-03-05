package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) DeleteResource(ctx context.Context, req *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	deleteCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err)
	}
	operator := operations.New(r.subscriber, r.resourceAggregateClient)
	deletedEvent, err := operator.DeleteResource(ctx, deleteCommand)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot delete resource: %v", err)
	}
	return pb.RAResourceDeletedEventToResponse(deletedEvent)
}

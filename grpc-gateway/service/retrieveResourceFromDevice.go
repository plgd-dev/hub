package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) RetrieveResourceFromDevice(ctx context.Context, req *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	retrieveCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource: %v", err)
	}
	operator := operations.New(r.subscriber, r.resourceAggregateClient)
	retrievedEvent, err := operator.RetrieveResource(ctx, retrieveCommand)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource: %v", err)
	}
	return pb.RAResourceRetrievedEventToResponse(retrievedEvent)
}

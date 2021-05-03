package service

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) RetrieveResourceFromDevice(ctx context.Context, req *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	retrieveCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource: %v", err))
	}
	retrievedEvent, err := r.resourceAggregateClient.SyncRetrieveResource(ctx, retrieveCommand)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource: %v", err))
	}
	return pb.RAResourceRetrievedEventToResponse(retrievedEvent)
}

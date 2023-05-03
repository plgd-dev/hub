package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func logAndReturnGetResourceFromDeviceError(err error) error {
	return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot retrieve resource: %v", err)
}

func (r *RequestHandler) GetResourceFromDevice(ctx context.Context, req *pb.GetResourceFromDeviceRequest) (*pb.GetResourceFromDeviceResponse, error) {
	retrieveCommand, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, logAndReturnGetResourceFromDeviceError(err)
	}
	retrievedEvent, err := r.resourceAggregateClient.SyncRetrieveResource(ctx, "*", retrieveCommand)
	if err != nil {
		return nil, logAndReturnGetResourceFromDeviceError(err)
	}
	err = commands.CheckEventContent(retrievedEvent)
	if err != nil {
		return nil, logAndReturnGetResourceFromDeviceError(err)
	}
	return &pb.GetResourceFromDeviceResponse{Data: retrievedEvent}, nil
}

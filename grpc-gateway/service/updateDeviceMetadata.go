package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
)

func (r *RequestHandler) UpdateDeviceMetadata(ctx context.Context, req *pb.UpdateDeviceMetadataRequest) (*pb.UpdateDeviceMetadataResponse, error) {
	correlationID := uuid.New()
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	shadowSynchronization := commands.ShadowSynchronization_DISABLED
	if req.GetShadowSynchronization() == pb.UpdateDeviceMetadataRequest_ENABLED {
		shadowSynchronization = commands.ShadowSynchronization_ENABLED
	}

	_, err := r.resourceAggregateClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      req.GetDeviceId(),
		CorrelationId: correlationID.String(),
		Update: &commands.UpdateDeviceMetadataRequest_ShadowSynchronization{
			ShadowSynchronization: shadowSynchronization,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})

	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update shadow synchronization of device %v: %v", req.GetDeviceId(), err))
	}
	return &pb.UpdateDeviceMetadataResponse{}, nil
}

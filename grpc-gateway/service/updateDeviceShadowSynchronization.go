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

func (r *RequestHandler) UpdateDeviceShadowSynchronization(ctx context.Context, req *pb.UpdateDeviceShadowSynchronizationRequest) (*pb.UpdateDeviceShadowSynchronizationResponse, error) {
	correlationID := uuid.New()
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}

	_, err := r.resourceAggregateClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId:      req.GetDeviceId(),
		CorrelationId: correlationID.String(),
		Update: &commands.UpdateDeviceMetadataRequest_ShadowSynchronization{
			ShadowSynchronization: &commands.ShadowSynchronization{
				Disabled: req.GetDisabled(),
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	})

	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update shadow synchronization of device %v: %v", req.GetDeviceId(), err))
	}
	return &pb.UpdateDeviceShadowSynchronizationResponse{}, nil
}

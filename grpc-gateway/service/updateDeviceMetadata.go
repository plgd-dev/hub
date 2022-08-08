package service

import (
	"context"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) UpdateDeviceMetadata(ctx context.Context, req *pb.UpdateDeviceMetadataRequest) (*pb.UpdateDeviceMetadataResponse, error) {
	updateMetadata, err := req.ToRACommand(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update device('%v') metadata: %v", req.GetDeviceId(), err))
	}
	metadataUpdated, err := r.resourceAggregateClient.SyncUpdateDeviceMetadata(ctx, "*", updateMetadata)
	if err != nil {
		return nil, log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot update device('%v') metadata: %v", req.GetDeviceId(), err))
	}
	return &pb.UpdateDeviceMetadataResponse{
		Data: metadataUpdated,
	}, nil
}

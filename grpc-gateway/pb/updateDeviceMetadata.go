package pb

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/peer"
)

func (req *UpdateDeviceMetadataRequest) ToRACommand(ctx context.Context) (*commands.UpdateDeviceMetadataRequest, error) {
	correlationUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}

	shadowSynchronization := commands.ShadowSynchronization_DISABLED
	if req.GetShadowSynchronization() == UpdateDeviceMetadataRequest_ENABLED {
		shadowSynchronization = commands.ShadowSynchronization_ENABLED
	}

	return &commands.UpdateDeviceMetadataRequest{
		DeviceId:      req.GetDeviceId(),
		CorrelationId: correlationUUID.String(),
		TimeToLive:    req.GetTimeToLive(),
		Update: &commands.UpdateDeviceMetadataRequest_ShadowSynchronization{
			ShadowSynchronization: shadowSynchronization,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

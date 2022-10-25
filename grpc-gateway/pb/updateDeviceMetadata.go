package pb

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
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

	twinEnabled := req.GetTwinEnabled()

	return &commands.UpdateDeviceMetadataRequest{
		DeviceId:      req.GetDeviceId(),
		CorrelationId: correlationUUID.String(),
		TimeToLive:    req.GetTimeToLive(),
		Update: &commands.UpdateDeviceMetadataRequest_TwinEnabled{
			TwinEnabled: twinEnabled,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

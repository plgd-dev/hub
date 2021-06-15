package pb

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/peer"
)

func (req *RetrieveResourceFromDeviceRequest) ToRACommand(ctx context.Context) (*commands.RetrieveResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}

	return &commands.RetrieveResourceRequest{
		ResourceId:        commands.ResourceIdFromString(req.GetResourceId()),
		CorrelationId:     correlationUUID.String(),
		ResourceInterface: req.GetResourceInterface(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

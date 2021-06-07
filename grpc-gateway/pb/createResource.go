package pb

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/peer"
)

func (req *CreateResourceRequest) ToRACommand(ctx context.Context) (*commands.CreateResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	return &commands.CreateResourceRequest{
		ResourceId:    req.GetResourceId(),
		CorrelationId: correlationUUID.String(),
		Content: &commands.Content{
			Data:              req.GetContent().GetData(),
			ContentType:       req.GetContent().GetContentType(),
			CoapContentFormat: -1,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

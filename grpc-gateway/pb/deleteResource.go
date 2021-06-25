package pb

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/peer"
)

func (req *DeleteResourceRequest) ToRACommand(ctx context.Context) (*commands.DeleteResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	connectionID := ""
	peer, ok := peer.FromContext(ctx)
	if ok {
		connectionID = peer.Addr.String()
	}
	href := req.GetResourceId().GetHref()
	if len(href) > 0 && href[0] != '/' {
		href = "/" + href
	}
	return &commands.DeleteResourceRequest{
		ResourceId:    commands.NewResourceID(req.GetResourceId().GetDeviceId(), href),
		CorrelationId: correlationUUID.String(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

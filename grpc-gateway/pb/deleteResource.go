package pb

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/peer"
)

func (req *DeleteResourceRequest) ToRACommand(ctx context.Context) (*commands.DeleteResourceRequest, error) {
	correlationUUID, err := uuid.NewRandom()
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
		TimeToLive:    req.GetTimeToLive(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
		ResourceInterface: req.GetResourceInterface(),
		Force:             req.GetForce(),
	}, nil
}

func (x *DeleteResourceResponse) SetData(data *events.ResourceDeleted) {
	if x == nil {
		return
	}
	x.Data = data
}

package pb

import (
	"context"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc/peer"
)

func (req *CreateResourceRequest) ToRACommand(ctx context.Context) (*commands.CreateResourceRequest, error) {
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

	return &commands.CreateResourceRequest{
		ResourceId:    commands.NewResourceID(req.GetResourceId().GetDeviceId(), href),
		CorrelationId: correlationUUID.String(),
		TimeToLive:    req.GetTimeToLive(),
		Content: &commands.Content{
			Data:              req.GetContent().GetData(),
			ContentType:       req.GetContent().GetContentType(),
			CoapContentFormat: -1,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
		Force: req.GetForce(),
	}, nil
}

func (x *CreateResourceResponse) SetData(data *events.ResourceCreated) {
	if x == nil {
		return
	}
	x.Data = data
}

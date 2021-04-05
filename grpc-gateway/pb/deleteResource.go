package pb

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
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
	return &commands.DeleteResourceRequest{
		ResourceId:    req.GetResourceId(),
		CorrelationId: correlationUUID.String(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: connectionID,
		},
	}, nil
}

func RAResourceDeletedEventToResponse(e *events.ResourceDeleted) (*DeleteResourceResponse, error) {
	content, err := EventContentToContent(e)
	if err != nil {
		return nil, err
	}
	return &DeleteResourceResponse{
		Content: content,
		Status:  RAStatus2Status(e.GetStatus()),
	}, nil
}

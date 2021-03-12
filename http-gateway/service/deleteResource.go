package service

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		writeError(w, fmt.Errorf("cannot create correlationID: %w", err))
		return
	}

	vars := mux.Vars(r)
	ctx := requestHandler.makeCtx(r)

	deleteCommand := &commands.DeleteResourceRequest{
		ResourceId:    commands.NewResourceID(vars[uri.DeviceIDKey], vars[uri.HrefKey]),
		CorrelationId: correlationUUID.String(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	}

	deletedEvent, err := requestHandler.raClient.SyncDeleteResource(ctx, deleteCommand)
	if err != nil {
		writeError(w, fmt.Errorf("cannot delete resource: %w", err))
		return
	}
	resp, err := pb.RAResourceDeletedEventToResponse(deletedEvent)
	if err != nil {
		writeError(w, fmt.Errorf("cannot delete resource: %w", err))
		return
	}

	var respBody interface{}
	err = client.DecodeContentWithCodec(client.GeneralMessageCodec{}, resp.GetContent().GetContentType(), resp.GetContent().GetData(), &respBody)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode response of deleted resource: %w", err))
		return
	}

	jsonResponseWriter(w, respBody)
}

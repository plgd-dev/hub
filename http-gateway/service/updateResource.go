package service

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"

	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) updateResource(w http.ResponseWriter, r *http.Request) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		writeError(w, fmt.Errorf("cannot create correlationID: %w", err))
		return
	}

	var body interface{}
	if err := json.ReadFrom(r.Body, &body); err != nil {
		writeError(w, fmt.Errorf("invalid json body: %w", err))
		return
	}

	vars := mux.Vars(r)
	interfaceQueryString := r.URL.Query().Get(uri.InterfaceQueryKey)
	ctx := requestHandler.makeCtx(r)

	data, err := cbor.Encode(body)
	if err != nil {
		writeError(w, fmt.Errorf("cannot encode to cbor: %w", err))
		return
	}

	updateCommand := &commands.UpdateResourceRequest{
		ResourceId:    commands.NewResourceID(vars[uri.DeviceIDKey], vars[uri.HrefKey]),
		CorrelationId: correlationUUID.String(),
		Content: &commands.Content{
			Data:              data,
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: -1,
		},
		ResourceInterface: interfaceQueryString,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	}

	updatedEvent, err := requestHandler.raClient.SyncUpdateResource(ctx, updateCommand)
	if err != nil {
		writeError(w, fmt.Errorf("cannot update resource: %w", err))
		return
	}
	resp, err := pb.RAResourceUpdatedEventToResponse(updatedEvent)
	if err != nil {
		writeError(w, fmt.Errorf("cannot update resource: %w", err))
		return
	}

	var respBody interface{}
	err = client.DecodeContentWithCodec(client.GeneralMessageCodec{}, resp.GetContent().GetContentType(), resp.GetContent().GetData(), &respBody)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode response of updated resource: %w", err))
		return
	}

	jsonResponseWriter(w, respBody)
}

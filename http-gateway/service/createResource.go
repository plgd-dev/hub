package service

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) createResource(w http.ResponseWriter, r *http.Request) {
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
	ctx := requestHandler.makeCtx(r)

	data, err := cbor.Encode(body)
	if err != nil {
		writeError(w, fmt.Errorf("cannot encode to cbor: %w", err))
		return
	}

	createCommand := &commands.CreateResourceRequest{
		ResourceId:    commands.NewResourceID(vars[uri.DeviceIDKey], vars[uri.HrefKey]),
		CorrelationId: correlationUUID.String(),
		Content: &commands.Content{
			Data:              data,
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: -1,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	}

	operator := operations.New(requestHandler.resourceSubscriber, requestHandler.raClient)
	createdEvent, err := operator.CreateResource(ctx, createCommand)
	if err != nil {
		writeError(w, fmt.Errorf("cannot create resource: %w", err))
		return
	}
	resp, err := pb.RAResourceCreatedEventToResponse(createdEvent)
	if err != nil {
		writeError(w, fmt.Errorf("cannot create resource: %w", err))
		return
	}

	var respBody interface{}
	err = client.DecodeContentWithCodec(client.GeneralMessageCodec{}, resp.GetContent().GetContentType(), resp.GetContent().GetData(), &respBody)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode response of created resource: %w", err))
		return
	}

	jsonResponseWriter(w, respBody)
}

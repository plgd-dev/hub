package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	Href := parseHref(vars[uri.HrefKey])
	interfaceQueryKeyString := r.URL.Query().Get(uri.InterfaceQueryKey)
	skipShadowQueryString := r.URL.Query().Get(uri.SkipShadowQueryKey)
	var getResourceFromDevice bool
	if interfaceQueryKeyString != "" {
		getResourceFromDevice = true
	}
	if skipShadowQueryString == "1" || strings.ToLower(skipShadowQueryString) == "true" {
		getResourceFromDevice = true
	}
	if getResourceFromDevice {
		requestHandler.getResourceFromDevice(w, r, interfaceQueryKeyString)
		return
	}

	ctx := requestHandler.makeCtx(r)
	var rep interface{}
	err := requestHandler.client.GetResource(ctx, vars[uri.DeviceIDKey], Href, &rep)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get resource: %w", err))
		return
	}

	jsonResponseWriter(w, rep)
}

func (requestHandler *RequestHandler) getResourceFromDevice(w http.ResponseWriter, r *http.Request, resourceInterface string) {
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

	retrieveCommand := &commands.RetrieveResourceRequest{
		ResourceId:    commands.NewResourceID(vars[uri.DeviceIDKey], vars[uri.HrefKey]),
		CorrelationId: correlationUUID.String(),

		ResourceInterface: interfaceQueryString,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	}

	operator := operations.New(requestHandler.resourceSubscriber, requestHandler.raClient)
	retrievedEvent, err := operator.RetrieveResource(ctx, retrieveCommand)
	if err != nil {
		writeError(w, fmt.Errorf("cannot retrieve resource: %w", err))
		return
	}
	resp, err := pb.RAResourceRetrievedEventToResponse(retrievedEvent)
	if err != nil {
		writeError(w, fmt.Errorf("cannot retrieve resource: %w", err))
		return
	}

	var respBody interface{}
	err = client.DecodeContentWithCodec(client.GeneralMessageCodec{}, resp.GetContent().GetContentType(), resp.GetContent().GetData(), &respBody)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode response of retrieved resource: %w", err))
		return
	}

	jsonResponseWriter(w, respBody)
}

func parseHref(linkQueryHref string) string {
	if idx := strings.IndexByte(linkQueryHref, '?'); idx >= 0 {
		return linkQueryHref[:idx]
	}
	return linkQueryHref
}

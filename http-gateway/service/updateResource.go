package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/http-gateway/uri"

	"github.com/go-ocf/kit/codec/json"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) updateResource(w http.ResponseWriter, r *http.Request) {
	var body interface{}
	if err := json.ReadFrom(r.Body, &body); err != nil {
		writeError(w, fmt.Errorf("invalid json body: %w", err))
		return
	}

	vars := mux.Vars(r)
	interfaceQueryString := r.URL.Query().Get(uri.InterfaceQueryKey)
	ctx := requestHandler.makeCtx(r)

	var response interface{}
	err := requestHandler.client.UpdateResource(ctx, vars[uri.DeviceIDKey], vars[uri.HrefKey], body, &response, client.WithInterface(interfaceQueryString))
	if err != nil {
		writeError(w, fmt.Errorf("cannot update resource: %w", err))
		return
	}

	jsonResponseWriter(w, response)
}

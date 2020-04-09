package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/http-gateway/uri"
	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) factoryResetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx, cancel := requestHandler.makeCtx(r)
	defer cancel()
	err := requestHandler.client.FactoryReset(ctx, vars[uri.DeviceIDKey])
	if err != nil {
		writeError(w, fmt.Errorf("cannot factory reset device: %w", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

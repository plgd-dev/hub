package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
)

func (requestHandler *RequestHandler) factoryResetDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := requestHandler.makeCtx(r)
	err := requestHandler.client.FactoryReset(ctx, vars[uri.DeviceIDKey])
	if err != nil {
		writeError(w, fmt.Errorf("cannot factory reset device: %w", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

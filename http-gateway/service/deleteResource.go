package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/http-gateway/uri"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) deleteResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	Href := parseHref(vars[uri.HrefKey])
	ctx := requestHandler.makeCtx(r)
	var rep interface{}
	err := requestHandler.client.DeleteResource(ctx, vars[uri.DeviceIDKey], Href, &rep)
	if err != nil {
		writeError(w, err)
		return
	}

	jsonResponseWriter(w, rep)
}

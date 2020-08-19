package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/http-gateway/uri"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	Href := parseHref(vars[uri.HrefKey])
	interfaceQueryKeyString := r.URL.Query().Get(uri.InterfaceQueryKey)
	skipShadowQueryString := r.URL.Query().Get(uri.SkipShadowQueryKey)
	ocfOpts := make([]client.GetOption, 0, 2)
	if interfaceQueryKeyString != "" {
		ocfOpts = append(ocfOpts, client.WithInterface(interfaceQueryKeyString))
	}
	if skipShadowQueryString == "1" || strings.ToLower(skipShadowQueryString) == "true" {
		ocfOpts = append(ocfOpts, client.WithSkipShadow())
	}
	ctx := requestHandler.makeCtx(r)
	var rep interface{}
	err := requestHandler.client.GetResource(ctx, vars[uri.DeviceIDKey], Href, &rep, ocfOpts...)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get resource: %w", err))
		return
	}

	jsonResponseWriter(w, rep)
}

func parseHref(linkQueryHref string) string {
	if idx := strings.IndexByte(linkQueryHref, '?'); idx >= 0 {
		return linkQueryHref[:idx]
	}
	return linkQueryHref
}

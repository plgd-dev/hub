package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-ocf/sdk/backend"

	"github.com/go-ocf/cloud/http-gateway/uri"

	"github.com/gorilla/mux"
)

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	resourceLinkHref := parseResourceLinkHref(vars[uri.ResourceLinkHrefKey])
	interfaceQueryKeyString := r.URL.Query().Get(uri.InterfaceQueryKey)
	skipShadowQueryString := r.URL.Query().Get(uri.SkipShadowQueryKey)
	ocfOpts := make([]backend.GetOption, 0, 2)
	if interfaceQueryKeyString != "" {
		ocfOpts = append(ocfOpts, backend.WithInterface(interfaceQueryKeyString))
	}
	if skipShadowQueryString == "1" || strings.ToLower(skipShadowQueryString) == "true" {
		ocfOpts = append(ocfOpts, backend.WithSkipShadow())
	}
	ctx, cancel := requestHandler.makeCtx(r)
	defer cancel()
	var rep interface{}
	err := requestHandler.client.GetResource(ctx, vars[uri.DeviceIDKey], resourceLinkHref, &rep, ocfOpts...)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get resource: %w", err))
		return
	}

	jsonResponseWriter(w, rep)
}

func parseResourceLinkHref(linkQueryHref string) string {
	if idx := strings.IndexByte(linkQueryHref, '?'); idx >= 0 {
		return linkQueryHref[:idx]
	}
	return linkQueryHref
}

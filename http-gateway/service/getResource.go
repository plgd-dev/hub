package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func (requestHandler *RequestHandler) getResourceFromShadow(w http.ResponseWriter, r *http.Request, resourceID string) {
	type Options struct {
		ResourceIdsFilter []string `url:"resourceIdsFilter"`
	}
	opt := Options{
		ResourceIdsFilter: []string{resourceID},
	}
	v, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get resource('%v') from shadow: %w", resourceID, err))
		return
	}
	r.URL.Path = uri.Resources
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)

	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		writeError(w, fmt.Errorf("cannot get resource('%v') from shadow: %w", resourceID, err))
	}, streamResponseKey, "data")
}

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	resourceHref := vars[uri.ResourceHrefKey]
	resourceID := commands.NewResourceID(deviceID, resourceHref).ToString()
	shadow := r.URL.Query().Get(uri.ShadowQueryKey)
	resourceInterface := r.URL.Query().Get(uri.ResourceInterfaceQueryKey)
	if (shadow == "" || strings.ToLower(shadow) == "true") && resourceInterface == "" {
		requestHandler.getResourceFromShadow(w, r, resourceID)
		return
	}

	requestHandler.mux.ServeHTTP(w, r)
}

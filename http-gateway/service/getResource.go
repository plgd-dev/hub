package service

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func (requestHandler *RequestHandler) getResourceFromShadow(w http.ResponseWriter, r *http.Request, resourceID string) {
	type Options struct {
		ResourceIDFilter []string `url:"resourceIdFilter"`
	}
	opt := Options{
		ResourceIDFilter: []string{resourceID},
	}
	v, err := query.Values(opt)
	if err != nil {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get resource('%v') from shadow: %v", resourceID, err))
		return
	}
	r.URL.Path = uri.Resources
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)

	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get resource('%v') from shadow: %v", resourceID, err))
	}, streamResponseKey)
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

package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
)

func (requestHandler *RequestHandler) getDeviceResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]

	type Options struct {
		DeviceIDFilter []string `url:"deviceIdFilter"`
	}
	opt := Options{
		DeviceIDFilter: []string{deviceID},
	}
	q, err := query.Values(opt)
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot get device('%v') resources: %w", deviceID, err))
		return
	}
	for key, values := range r.URL.Query() {
		if key == uri.TypeFilterQueryKey {
			for _, v := range values {
				q.Add(key, v)
			}
		}
	}
	r.URL.Path = uri.Resources
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

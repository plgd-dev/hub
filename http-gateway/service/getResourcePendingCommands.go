package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func (requestHandler *RequestHandler) getResourcePendingCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	href := vars[uri.ResourceHrefKey]
	resourceID := commands.NewResourceID(deviceID, href).ToString()

	type Options struct {
		ResourceIdFilter []string `url:"resourceIdFilter"`
	}
	opt := Options{
		ResourceIdFilter: []string{resourceID},
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get resource('%v') pending commands: %w", resourceID, err))
		return
	}
	for key, values := range r.URL.Query() {
		switch key {
		case uri.CommandFilterQueryKey:
			for _, v := range values {
				q.Add(key, v)
			}
		}
	}
	r.URL.Path = uri.PendingCommands
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

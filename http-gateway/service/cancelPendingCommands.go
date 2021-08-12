package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func (requestHandler *RequestHandler) CancelPendingCommands(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	href := vars[uri.ResourceHrefKey]
	correlationID := vars[uri.CorrelationIDKey]
	resourceID := commands.NewResourceID(deviceID, href).ToString()

	type Options struct {
		DeviceID      string   `url:"resourceId.deviceId"`
		Href          string   `url:"resourceId.href"`
		CorrelationId []string `url:"correlationIdFilter,omitempty"`
	}
	opt := Options{
		DeviceID: deviceID,
		Href:     href,
	}
	if correlationID != "" {
		opt.CorrelationId = append(opt.CorrelationId, correlationID)
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot cancel resource('%v') commands: %w", resourceID, err))
		return
	}
	for key, values := range r.URL.Query() {
		switch key {
		case uri.CorrelationIdFilterQueryKey:
			for _, v := range values {
				q.Add(key, v)
			}
		}
	}
	r.URL.Path = uri.PendingCommands
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

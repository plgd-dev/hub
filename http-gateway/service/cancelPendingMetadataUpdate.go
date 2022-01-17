package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/jtacoma/uritemplates"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
)

func (requestHandler *RequestHandler) cancelPendingMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	correlationID := vars[uri.CorrelationIDKey]

	cannotCancelError := func(err error) error {
		return fmt.Errorf("cannot cancel device('%v') metadata update: %w", deviceID, err)
	}

	type Options struct {
		CorrelationID string `url:"correlationIdFilter"`
	}
	opt := Options{
		CorrelationID: correlationID,
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, cannotCancelError(err))
		return
	}
	tmp, err := uritemplates.Parse(uri.AliasDevicePendingMetadataUpdates)
	if err != nil {
		writeError(w, cannotCancelError(err))
		return
	}
	urlPath, err := tmp.Expand(map[string]interface{}{
		uri.DeviceIDKey: deviceID,
	})
	if err != nil {
		writeError(w, cannotCancelError(err))
		return
	}
	r.URL.Path = urlPath
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

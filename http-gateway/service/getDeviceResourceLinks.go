package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
)

const streamResponseKey = "result"

func (requestHandler *RequestHandler) getDeviceResourceLinks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	type Options struct {
		DeviceIdsFilter []string `url:"deviceIdsFilter"`
	}
	opt := Options{
		DeviceIdsFilter: []string{deviceID},
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get device('%v') resource links: %w", deviceID, err))
		return
	}
	for key, values := range r.URL.Query() {
		switch key {
		case uri.TypeFilterQueryKey:
			for _, v := range values {
				q.Add(key, v)
			}
		}
	}
	r.URL.Path = uri.ResourceLinks
	r.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)

	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		writeError(w, fmt.Errorf("cannot get device('%v') resource links: %w", deviceID, err))
	}, streamResponseKey)
}

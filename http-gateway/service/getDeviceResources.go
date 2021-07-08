package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
)

type notFoundResponseWrapperWriter struct {
	w                 http.ResponseWriter
	Code              int
	DataInBody        bool
	WriteHeaderCalled bool
}

func newNotFoundWrapperWriter(w http.ResponseWriter) *notFoundResponseWrapperWriter {
	return &notFoundResponseWrapperWriter{
		w: w,
	}
}

func (w *notFoundResponseWrapperWriter) Write(b []byte) (int, error) {
	if len(b) > 0 {
		if !w.DataInBody {
			w.DataInBody = true
			w.WriteHeader(w.Code)
		}
	}
	return w.w.Write(b)
}

func (w *notFoundResponseWrapperWriter) Header() http.Header {
	return w.w.Header()
}

func (w *notFoundResponseWrapperWriter) WriteHeader(code int) {
	w.Code = code
	if w.DataInBody && !w.WriteHeaderCalled && w.Code != 0 {
		w.w.WriteHeader(w.Code)
		w.WriteHeaderCalled = true
	}
}

func (w *notFoundResponseWrapperWriter) Flush() {
	f, ok := w.w.(http.Flusher)
	if ok && w.DataInBody {
		f.Flush()
	}
}

func (requestHandler *RequestHandler) getDeviceResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]

	type Options struct {
		DeviceIdFilter []string `url:"deviceIdFilter"`
	}
	opt := Options{
		DeviceIdFilter: []string{deviceID},
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get device('%v') resources: %w", deviceID, err))
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
	r.URL.Path = uri.Resources
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

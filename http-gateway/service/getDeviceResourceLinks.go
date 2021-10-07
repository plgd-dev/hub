package service

import (
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

const streamResponseKey = "result"

func (requestHandler *RequestHandler) getDeviceResourceLinks(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v') resource links: %v", deviceID, err))
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
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v') resource links: %v", deviceID, err))
	}, streamResponseKey)
}

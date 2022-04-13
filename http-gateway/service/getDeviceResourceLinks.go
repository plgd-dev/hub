package service

import (
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

const streamResponseKey = "result"

func (requestHandler *RequestHandler) getDeviceResourceLinks(w http.ResponseWriter, r *http.Request) {
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
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v') resource links: %v", deviceID, err))
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
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v') resource links: %v", deviceID, err))
	}, streamResponseKey)
}

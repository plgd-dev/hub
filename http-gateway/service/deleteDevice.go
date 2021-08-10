package service

import (
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (requestHandler *RequestHandler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	type Options struct {
		DeviceIdFilter []string `url:"deviceIdFilter"`
	}
	opt := Options{
		DeviceIdFilter: []string{deviceID},
	}
	v, err := query.Values(opt)
	if err != nil {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete device('%v'): %v", deviceID, err))
		return
	}
	r.URL.Path = uri.Devices
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)

	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete device('%v'): %v", deviceID, err))
	}, streamResponseKey)
}

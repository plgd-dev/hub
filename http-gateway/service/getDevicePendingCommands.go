package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
)

func (requestHandler *RequestHandler) getDevicePendingCommands(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, fmt.Errorf("cannot get device('%v') pending commands: %w", deviceID, err))
		return
	}
	for key, values := range r.URL.Query() {
		switch key {
		case uri.CommandFilterQueryKey:
			for _, v := range values {
				if v == pb.GetPendingCommandsRequest_DEVICE_METADATA_UPDATE.String() {
					continue
				}
				q.Add(key, v)
			}
		case uri.TypeFilterQueryKey:
			for _, v := range values {
				q.Add(key, v)
			}
		}
	}
	if q.Get(uri.CommandFilterQueryKey) == "" {
		q.Add(uri.CommandFilterQueryKey, pb.GetPendingCommandsRequest_RESOURCE_CREATE.String())
		q.Add(uri.CommandFilterQueryKey, pb.GetPendingCommandsRequest_RESOURCE_RETRIEVE.String())
		q.Add(uri.CommandFilterQueryKey, pb.GetPendingCommandsRequest_RESOURCE_UPDATE.String())
		q.Add(uri.CommandFilterQueryKey, pb.GetPendingCommandsRequest_RESOURCE_DELETE.String())
	}
	r.URL.Path = uri.PendingCommands
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

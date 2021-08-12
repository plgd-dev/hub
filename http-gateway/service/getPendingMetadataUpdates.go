package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
)

func (requestHandler *RequestHandler) getPendingMetadataUpdates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]

	type options struct {
		DeviceIdFilter []string `url:"deviceIdFilter"`
	}
	opt := options{
		DeviceIdFilter: []string{deviceID},
	}
	q, err := query.Values(opt)
	if err != nil {
		writeError(w, fmt.Errorf("cannot get device('%v') pending metadata updates: %w", deviceID, err))
		return
	}
	q.Add(uri.CommandFilterQueryKey, pb.GetPendingCommandsRequest_DEVICE_METADATA_UPDATE.String())
	r.URL.Path = uri.PendingCommands
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

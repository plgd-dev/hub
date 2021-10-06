package service

import (
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/v2/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/v2/http-gateway/uri"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/commands"
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
				if v == pb.GetPendingCommandsRequest_DEVICE_METADATA_UPDATE.String() {
					continue
				}
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

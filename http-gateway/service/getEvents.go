package service

import (
	"net/http"
	"strconv"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (requestHandler *RequestHandler) getEvents(w http.ResponseWriter, r *http.Request) {
	var timestamp int64
	var err error
	t := r.URL.Query().Get(uri.TimestampFilterQueryKey)
	if t != "" {
		timestamp, err = strconv.ParseInt(t, 10, 64)
		if err != nil {
			serverMux.WriteError(w, status.Errorf(codes.InvalidArgument, "failed to parse timestamp %v: %v", t, err))
			return
		}
	}

	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	href := vars[uri.ResourceHrefKey]
	resourceID := commands.NewResourceID(deviceID, href).ToString()
	type Options struct {
		DeviceIDFilter   []string `url:"deviceIdFilter,omitempty"`
		ResourceIDFilter []string `url:"resourceIdFilter,omitempty"`
		TimestampFilter  int64    `url:"timestampFilter,omitempty"`
	}
	opt := Options{}
	if resourceID != "" {
		opt.ResourceIDFilter = append(opt.ResourceIDFilter, resourceID)
	} else {
		if deviceID != "" {
			opt.DeviceIDFilter = append(opt.DeviceIDFilter, deviceID)
		}
	}
	if timestamp != 0 {
		opt.TimestampFilter = timestamp
	}
	q, err := query.Values(opt)
	if err != nil {
		serverMux.WriteError(w,
			status.Errorf(codes.InvalidArgument,
				"cannot get events (deviceId: %v, resourceId: %v, timestamp: %v): %v",
				deviceID, resourceID, timestamp, err,
			),
		)
		return
	}
	r.URL.Path = uri.Events
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}

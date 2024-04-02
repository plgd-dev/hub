package service

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (requestHandler *RequestHandler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID, rec, err := requestHandler.serveDevicesRequest(r)
	if err != nil {
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete device('%v'): %v", deviceID, err))
		return
	}
	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot delete device('%v'): %v", deviceID, err))
	}, streamResponseKey)
}

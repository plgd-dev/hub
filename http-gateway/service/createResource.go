package service

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func (requestHandler *RequestHandler) createResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	href := vars[uri.ResourceHrefKey]

	contentType := r.Header.Get(uri.ContentTypeHeaderKey)
	if contentType == uri.ApplicationProtoJsonContentType {
		requestHandler.mux.ServeHTTP(w, r)
		return
	}

	newBody, err := createContentBody(r.Body)
	if err != nil {
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot create resource('%v%v'): %v", deviceID, href, err))
		return
	}

	r.Body = newBody
	requestHandler.mux.ServeHTTP(w, r)
}

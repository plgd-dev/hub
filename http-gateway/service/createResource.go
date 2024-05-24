package service

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"google.golang.org/grpc/codes"
)

func (requestHandler *RequestHandler) createResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	href := vars[uri.ResourceHrefKey]

	contentType := r.Header.Get(pkgHttp.ContentTypeHeaderKey)
	if contentType == pkgHttp.ApplicationProtoJsonContentType {
		requestHandler.mux.ServeHTTP(w, r)
		return
	}

	newBody, err := createContentBody(r.Body)
	if err != nil {
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "cannot create resource('%v%v'): %v", deviceID, href, err))
		return
	}

	r.Body = newBody
	requestHandler.mux.ServeHTTP(w, r)
}

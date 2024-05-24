package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

func createContentBody(body io.ReadCloser) (io.ReadCloser, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	req := commands.Content{
		ContentType:       message.AppJSON.String(),
		CoapContentFormat: int32(message.AppJSON),
		Data:              data,
	}
	reqData, err := protojson.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal to protojson: %w", err)
	}

	return io.NopCloser(bytes.NewReader(reqData)), nil
}

func (requestHandler *RequestHandler) updateResource(w http.ResponseWriter, r *http.Request) {
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
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource('/%v%v'): %v", deviceID, href, err))
		return
	}

	r.Body = newBody
	rec := httptest.NewRecorder()
	onlyContent := r.URL.Query().Get(uri.OnlyContentQueryKey)
	requestHandler.mux.ServeHTTP(rec, r)
	allowEmptyContent := false
	if parseBoolQuery(onlyContent) {
		allowEmptyContent = requestHandler.filterOnlyContent(rec, "data", "content")
	}
	toSimpleResponse(w, rec, allowEmptyContent, func(w http.ResponseWriter, err error) {
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource('/%v%v') from the device: %v", deviceID, href, err))
	}, streamResponseKey)
}

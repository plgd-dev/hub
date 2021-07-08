package service

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

func createContentBody(body io.ReadCloser) (io.ReadCloser, error) {
	data, err := ioutil.ReadAll(body)
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

	return ioutil.NopCloser(bytes.NewReader(reqData)), nil

}

func (requestHandler *RequestHandler) updateResource(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot update resource('%v%v'): %v", deviceID, href, err))
		return
	}

	r.Body = newBody
	requestHandler.mux.ServeHTTP(w, r)
}

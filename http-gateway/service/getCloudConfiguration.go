package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/kit/codec/json"
	"google.golang.org/protobuf/encoding/protojson"
)

func getAccept(r *http.Request) string {
	accept := r.Header.Get(uri.AcceptHeaderKey)
	if accept != "" {
		return accept
	}
	accept = r.Header.Get(strings.ToLower(uri.AcceptHeaderKey))
	if accept != "" {
		return accept
	}
	return r.URL.Query().Get(uri.AcceptQueryKey)
}

func (requestHandler *RequestHandler) getCloudConfiguration(w http.ResponseWriter, r *http.Request) {
	accept := getAccept(r)
	if accept == uri.ApplicationProtoJsonContentType {
		requestHandler.mux.ServeHTTP(w, r)
		return
	}
	resp, err := requestHandler.client.GrpcGatewayClient().GetClientConfiguration(r.Context(), &pb.ClientConfigurationRequest{})
	if err != nil {
		writeError(w, fmt.Errorf("cannot get cloud configuration: %w", err))
		return
	}
	v := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	data, err := v.Marshal(resp)
	if err != nil {
		writeError(w, fmt.Errorf("cannot marshal cloud configuration: %w", err))
		return
	}
	var decoded map[string]interface{}
	err = json.Decode(data, &decoded)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode cloud configuration: %w", err))
		return
	}
	for key, value := range decoded {
		if s, ok := value.(string); ok {
			num, err := strconv.ParseInt(s, 10, 64)
			if err == nil {
				decoded[key] = num
			}
		}
	}
	if err := jsonResponseWriter(w, decoded); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

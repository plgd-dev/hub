package service

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/json"
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

func decodeHubConfiguration(data []byte) (map[string]interface{}, error) {
	var decoded map[string]interface{}
	err := json.Decode(data, &decoded)
	if err != nil {
		return nil, err
	}
	for key, value := range decoded {
		if s, ok := value.(string); ok {
			num, err := strconv.ParseInt(s, 10, 64)
			if err == nil {
				decoded[key] = num
				continue
			}
			hostQuery := strings.SplitN(s, "?", 2)
			if len(hostQuery) < 2 {
				// we cannot call QueryUnescape over schema otherwise "coaps+tcp" will be unescaped to "coaps tcp".
				continue
			}
			unescaped, err := url.QueryUnescape(hostQuery[1])
			if err == nil {
				query := bytes.ReplaceAll([]byte(unescaped), []byte("\\u003c"), []byte("<"))
				query = bytes.ReplaceAll(query, []byte("\\u003e"), []byte(">"))
				query = bytes.ReplaceAll(query, []byte("\\u0026"), []byte("&"))
				hostQuery[1] = string(query)
				decoded[key] = strings.Join(hostQuery, "?")
				continue
			}
		}
	}
	return decoded, nil
}

func (requestHandler *RequestHandler) getHubConfiguration(w http.ResponseWriter, r *http.Request) {
	accept := getAccept(r)
	resp, err := requestHandler.client.GrpcGatewayClient().GetHubConfiguration(r.Context(), &pb.HubConfigurationRequest{})
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot get hub configuration: %w", err))
		return
	}
	resp.HttpGatewayAddress = requestHandler.config.UI.WebConfiguration.HTTPGatewayAddress
	resp.DeviceOauthClient = requestHandler.config.UI.WebConfiguration.DeviceOAuthClient.ToProto()
	resp.WebOauthClient = requestHandler.config.UI.WebConfiguration.WebOAuthClient.ToProto()
	resp.Ui = &pb.UIConfiguration{
		Visibility: requestHandler.config.UI.WebConfiguration.Visibility.ToProto(),
	}
	if accept == uri.ApplicationProtoJsonContentType {
		m := serverMux.NewJsonpbMarshaler()
		w.Header().Set(contentTypeHeaderKey, uri.ApplicationProtoJsonContentType)
		w.WriteHeader(http.StatusOK)
		if err := m.NewEncoder(w).Encode(resp); err != nil {
			log.Errorf("failed to write response: %v", err)
		}
		return
	}
	v := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	data, err := v.Marshal(resp)
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot marshal cloud configuration: %w", err))
		return
	}
	decoded, err := decodeHubConfiguration(data)
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot decode cloud configuration: %w", err))
		return
	}

	if err := jsonResponseWriter(w, decoded); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

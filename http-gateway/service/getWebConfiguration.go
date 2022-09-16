package service

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

func (requestHandler *RequestHandler) getWebConfiguration(w http.ResponseWriter, r *http.Request) {
	resp, err := requestHandler.client.GrpcGatewayClient().GetHubConfiguration(r.Context(), &pb.HubConfigurationRequest{})
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot get hub configuration: %w", err))
		return
	}
	cfg := requestHandler.config.UI.WebConfiguration
	cfg.Authority = resp.GetAuthority()
	if err := jsonResponseWriter(w, cfg); err != nil {
		log.Errorf("failed to write response: %w", err)
	}
}

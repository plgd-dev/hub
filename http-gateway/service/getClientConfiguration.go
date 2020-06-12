package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
)

func (requestHandler *RequestHandler) getClientConfiguration(w http.ResponseWriter, r *http.Request) {
	ctx := requestHandler.makeCtx(r)
	resp, err := requestHandler.client.GrpcGatewayClient().GetClientConfiguration(ctx, &pb.ClientConfigurationRequest{})
	if err != nil {
		writeError(w, fmt.Errorf("cannot reboot device: %w", err))
		return
	}
	jsonResponseWriter(w, resp)
}

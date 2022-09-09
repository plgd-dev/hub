package http

import (
	"context"
	"fmt"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc" //	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
	mux    *runtime.ServeMux
}

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, certificateAuthorityServer *grpcService.CertificateAuthorityServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterCertificateAuthorityServer(ch, certificateAuthorityServer)
	grpcClient := pb.NewCertificateAuthorityClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterCertificateAuthorityHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register certificate-authority handler: %w", err)
	}
	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	grpcService "github.com/plgd-dev/hub/v2/m2m-oauth-server/service/grpc"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config                *Config
	m2mOAuthServiceServer *grpcService.M2MOAuthServiceServer
	mux                   *runtime.ServeMux
}

// NewRequestHandler returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, m2mOAuthServiceServer *grpcService.M2MOAuthServiceServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config:                config,
		mux:                   serverMux.New(),
		m2mOAuthServiceServer: m2mOAuthServiceServer,
	}

	r.HandleFunc(uri.OpenIDConfiguration, requestHandler.getOpenIDConfiguration).Methods(http.MethodGet)
	r.HandleFunc(uri.JWKs, requestHandler.getJWKs).Methods(http.MethodGet)
	r.HandleFunc(uri.Token, requestHandler.postToken).Methods(http.MethodPost)

	ch := new(inprocgrpc.Channel)
	pb.RegisterM2MOAuthServiceServer(ch, m2mOAuthServiceServer)
	grpcClient := pb.NewM2MOAuthServiceClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterM2MOAuthServiceHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register m2m-oauth-server handler: %w", err)
	}
	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

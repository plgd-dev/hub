package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/integration-service/pb"
	grpcService "github.com/plgd-dev/hub/v2/integration-service/service/grpc"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
	mux    *runtime.ServeMux
}

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, integrationServiceServer *grpcService.IntegrationServiceServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
	}

	r.HandleFunc(AliasConfig, requestHandler.getConfig).Methods(http.MethodGet)

	ch := new(inprocgrpc.Channel)
	pb.RegisterIntegrationServiceServer(ch, integrationServiceServer)
	grpcClient := pb.NewIntegrationServiceClient(ch)

	// register grpc-proxy handler
	if err := pb.RegisterIntegrationServiceHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register integration service handler: %w", err)
	}

	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

package http

import (
	"context"
	"fmt"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	grpcService "github.com/plgd-dev/hub/v2/snippet-service/service/grpc"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
	mux    *runtime.ServeMux
}

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, snippetServiceServer *grpcService.SnippetServiceServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterSnippetServiceServer(ch, snippetServiceServer)
	grpcClient := pb.NewSnippetServiceClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterSnippetServiceHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register snippet-service handler: %w", err)
	}
	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

package http

import (
	"context"
	"fmt"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/snapshot-service/pb"
	grpcService "github.com/plgd-dev/hub/v2/snapshot-service/service/grpc"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
	mux    *runtime.ServeMux
}

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, snapshotServiceServer *grpcService.SnapshotServiceServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterSnapshotServiceServer(ch, snapshotServiceServer)
	grpcClient := pb.NewSnapshotServiceClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterSnapshotServiceHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register snapshot-service handler: %w", err)
	}
	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

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

/*
// TODO: the GRPC query parser doesn't seem to support oneOf fields, so we have to manually encode and decode the query
func (requestHandler *RequestHandler) getConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	// /api/v1/configuration/{id}?version=latest -> rpc GetConfigurations + IDFilter{IDFilter_Latest}
	// /api/v1/configuration/{id}?version=all -> rpc GetConfigurations + IDFilter{IDFilter_All}
	// /api/v1/configuration/{id}?version={version} -> rpc GetConfigurations + IDFilter{IDFilter_Version{version}}
	vars := mux.Vars(r)
	configurationID := vars[IDKey]

	versionStr := r.URL.Query().Get(VersionQueryKey)
	if versionStr != "" && versionStr != "all" && versionStr == "latest" {
		var err error
		_, err = strconv.ParseUint(versionStr, 10, 64)
		if err != nil {
			serverMux.WriteError(w, fmt.Errorf("invalid configuration('%v') version: %w", configurationID, err))
			return
		}
	}

	type Options struct {
		HTTPIDFilter []string `url:"httpIdFilter"`
	}
	opt := &Options{
		HTTPIDFilter: []string{configurationID + "/" + versionStr},
	}
	q, err := query.Values(opt)
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("invalid configuration('%v') version: %w", configurationID, err))
		return
	}

	r.URL.Path = Configurations
	r.URL.RawQuery = q.Encode()
	requestHandler.mux.ServeHTTP(w, r)
}
*/

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, snippetServiceServer *grpcService.SnippetServiceServer) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
	}

	// Aliases
	// r.HandleFunc(AliasConfigurations, requestHandler.getConfigurationVersion).Methods(http.MethodGet)

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

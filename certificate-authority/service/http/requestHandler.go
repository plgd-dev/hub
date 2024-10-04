package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcService "github.com/plgd-dev/hub/v2/certificate-authority/service/grpc"
	"github.com/plgd-dev/hub/v2/certificate-authority/service/uri"
	"github.com/plgd-dev/hub/v2/certificate-authority/store"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
	mux    *runtime.ServeMux

	cas   *grpcService.CertificateAuthorityServer
	store store.Store
}

// NewHTTP returns HTTP handler
func NewRequestHandler(config *Config, r *mux.Router, cas *grpcService.CertificateAuthorityServer, s store.Store) (*RequestHandler, error) {
	requestHandler := &RequestHandler{
		config: config,
		mux:    serverMux.New(),
		cas:    cas,
		store:  s,
	}

	if config.CRLEnabled {
		r.HandleFunc(uri.SigningRevocationList, requestHandler.revocationList).Methods(http.MethodGet)
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterCertificateAuthorityServer(ch, cas)
	grpcClient := pb.NewCertificateAuthorityClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterCertificateAuthorityHandlerClient(context.Background(), requestHandler.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register certificate-authority handler: %w", err)
	}
	r.PathPrefix("/").Handler(requestHandler.mux)

	return requestHandler, nil
}

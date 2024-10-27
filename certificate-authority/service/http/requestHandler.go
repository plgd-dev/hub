package http

import (
	"context"
	"errors"
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

// requestHandler for handling incoming request
type requestHandler struct {
	config *Config
	mux    *runtime.ServeMux

	cas   *grpcService.CertificateAuthorityServer
	store store.Store
}

// NewHTTP returns HTTP handler
func newRequestHandler(config *Config, r *mux.Router, cas *grpcService.CertificateAuthorityServer, s store.Store) (*requestHandler, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if r == nil {
		return nil, errors.New("router cannot be nil")
	}
	if cas == nil {
		return nil, errors.New("certificate authority server cannot be nil")
	}
	if s == nil {
		return nil, errors.New("store cannot be nil")
	}
	rh := &requestHandler{
		config: config,
		mux:    serverMux.New(),
		cas:    cas,
		store:  s,
	}

	if config.CRLEnabled {
		r.HandleFunc(uri.SigningRevocationList, rh.revocationList).Methods(http.MethodGet)
	}

	ch := new(inprocgrpc.Channel)
	pb.RegisterCertificateAuthorityServer(ch, cas)
	grpcClient := pb.NewCertificateAuthorityClient(ch)
	// register grpc-proxy handler
	if err := pb.RegisterCertificateAuthorityHandlerClient(context.Background(), rh.mux, grpcClient); err != nil {
		return nil, fmt.Errorf("failed to register certificate-authority handler: %w", err)
	}
	r.PathPrefix("/").Handler(rh.mux)

	return rh, nil
}

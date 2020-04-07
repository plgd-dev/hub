package service

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/smallstep/certificates/acme"
	acmeAPI "github.com/smallstep/certificates/acme/api"
	"github.com/smallstep/nosql"
)

func NewHandler(FQDN string, port uint16, signers []Signer) (http.Handler, error) {
	// Using chi as the main router
	mux := chi.NewRouter()
	handler := http.Handler(mux)
	// A good base middleware stack
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	sigs := make(map[string]Signer)
	for _, s := range signers {
		sigs[s.ID()] = s
	}

	auth := authority{
		signers: sigs,
	}

	db, err := nosql.New("badger", "acmeDB")
	if err != nil {
		return nil, err
	}

	dns := FQDN + ":" + strconv.FormatUint(uint64(port), 10)
	prefix := "acme"
	acmeAuth := acme.NewAuthority(db, dns, prefix, &auth)
	acmeRouterHandler := acmeAPI.New(acmeAuth)
	mux.Route("/"+prefix, func(r chi.Router) {
		acmeRouterHandler.Route(r)
	})

	return handler, nil
}

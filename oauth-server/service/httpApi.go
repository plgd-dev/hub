package service

import (
	"net/http"
	"net/http/httputil"

	"github.com/plgd-dev/cloud/oauth-server/uri"
	"github.com/plgd-dev/kit/log"

	router "github.com/gorilla/mux"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	config *Config
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(config *Config) *RequestHandler {
	return &RequestHandler{
		config: config,
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := httputil.DumpRequest(r, false)
		if err != nil {
			log.Infof("Request: %v %v", r.Method, r.RequestURI)
		} else {
			log.Infof("Request: %v", string(data))
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// NewHTTP returns HTTP server
func NewHTTP(requestHandler *RequestHandler) *http.Server {
	r := router.NewRouter()
	r.Use(loggingMiddleware)
	r.StrictSlash(true)

	// get JWKs
	r.HandleFunc(uri.JWKs, requestHandler.getJWKs).Methods(http.MethodGet)

	return &http.Server{Handler: r}
}

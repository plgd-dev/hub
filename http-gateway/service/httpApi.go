package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pbCA "github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"

	"github.com/google/uuid"
	router "github.com/gorilla/mux"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	client   *client.Client
	config   *Config
	caClient pbCA.CertificateAuthorityClient
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(config *Config, client *client.Client, caClient pbCA.CertificateAuthorityClient) *RequestHandler {
	return &RequestHandler{
		client:   client,
		caClient: caClient,
		config:   config,
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

type logger struct{}

func (logger) Warnln(v ...interface{})  { fmt.Printf("%v\n", v) }
func (logger) Debugln(v ...interface{}) { fmt.Printf("%v\n", v) }

// NewHTTP returns HTTP server
func NewHTTP(requestHandler *RequestHandler, authInterceptor kitHttp.Interceptor) *http.Server {
	r := router.NewRouter()
	r.Use(loggingMiddleware)
	r.Use(kitHttp.CreateAuthMiddleware(authInterceptor, func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, fmt.Errorf("cannot access to %v: %w", r.RequestURI, err))
	}))
	r.StrictSlash(true)

	// certifica authority sign
	r.HandleFunc(uri.CertificaAuthoritySign, requestHandler.signCertificate).Methods(http.MethodPost)

	mux := runtime.NewServeMux()
	pb.RegisterGrpcGatewayHandlerClient(context.Background(), mux, requestHandler.client.GrpcGatewayClient())

	// ws
	ws := wsproxy.WebsocketProxy(mux, wsproxy.WithRequestMutator(func(incoming, outgoing *http.Request) *http.Request {
		outgoing.Method = http.MethodPost
		return outgoing
	}), wsproxy.WithLogger(logger{}))
	r.PathPrefix(uri.APIWS + "/").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		ws.ServeHTTP(rw, r)
	})

	// api
	r.Handle(uri.ClientConfiguration, mux)
	r.PathPrefix(uri.API + "/").HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(rw, r)
	})

	// serve www directory
	if requestHandler.config.UI.Enabled {
		r.HandleFunc(uri.OAuthConfiguration, requestHandler.getOAuthConfiguration).Methods(http.MethodGet)
		fs := http.FileServer(http.Dir(requestHandler.config.UI.Directory))
		r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := httptest.NewRecorder()
			fs.ServeHTTP(c, r)
			if c.Code == http.StatusNotFound {
				c = httptest.NewRecorder()
				r.URL.Path = "/"
				fs.ServeHTTP(c, r)
			}
			for k, v := range c.HeaderMap {
				w.Header().Set(k, strings.Join(v, ""))
			}
			w.WriteHeader(c.Code)
			c.Body.WriteTo(w)
		}))
	}

	return &http.Server{Handler: r}
}

func (requestHandler *RequestHandler) makeCtx(r *http.Request) context.Context {
	token := getAccessToken(r.Header)
	return kitNetGrpc.CtxWithToken(r.Context(), token)
}

func getAccessToken(h http.Header) string {
	accessToken := h.Get("Authorization")
	if len(accessToken) < 7 {
		return ""
	}
	return accessToken[7:]
}

func getCorrelationID(h http.Header) string {
	correlationID := h.Get("Correlation-ID")
	if correlationID == "" {
		return uuid.New().String()
	}
	return correlationID
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/google/uuid"
	pbCA "github.com/plgd-dev/cloud/certificate-authority/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/client"

	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	kitHttp "github.com/plgd-dev/kit/net/http"

	router "github.com/gorilla/mux"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	client   *client.Client
	caClient pbCA.CertificateAuthorityClient
	config   *Config
	manager  *ObservationManager
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(client *client.Client, caClient pbCA.CertificateAuthorityClient, config *Config, manager *ObservationManager) *RequestHandler {
	return &RequestHandler{
		client:   client,
		config:   config,
		manager:  manager,
		caClient: caClient,
	}
}

func resourceMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI, uri.Devices)
	if len(paths) > 1 {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[uri.DeviceIDKey] = paths[0]
		rm.Vars[uri.HrefKey] = "/" + strings.Join(paths[1:], "/")
		return true
	}
	return false
}

func wsResourceMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI, uri.WSDevices)
	if len(paths) > 1 {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[uri.DeviceIDKey] = paths[0]
		rm.Vars[uri.HrefKey] = "/" + strings.Join(paths[1:], "/")
		return true
	}
	return false
}

func splitDevicePath(requestURI, prefix string) []string {
	p := kitHttp.CanonicalHref(requestURI)
	p = strings.TrimPrefix(p, prefix) // remove core prefix
	p = strings.TrimLeft(p, "/")
	return strings.Split(p, "/")
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
func NewHTTP(requestHandler *RequestHandler, authInterceptor kitHttp.Interceptor) *http.Server {
	r := router.NewRouter()
	r.Use(loggingMiddleware)
	r.Use(kitHttp.CreateAuthMiddleware(authInterceptor, func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, fmt.Errorf("cannot process request on %v: %w", r.RequestURI, err))
	}))
	r.StrictSlash(true)

	// client configuration
	r.HandleFunc(uri.ClientConfiguration, requestHandler.getClientConfiguration).Methods(http.MethodGet)

	// certifica authority sign
	r.HandleFunc(uri.CertificaAuthoritySign, requestHandler.signCertificate).Methods(http.MethodPost)

	// devices
	r.HandleFunc(uri.Devices, requestHandler.getDevices).Methods(http.MethodGet)
	r.HandleFunc(uri.Device, requestHandler.getDevice).Methods(http.MethodGet)

	//maintenance
	r.HandleFunc(uri.DeviceReboot, requestHandler.rebootDevice).Methods(http.MethodPost)
	r.HandleFunc(uri.DeviceFactoryReset, requestHandler.factoryResetDevice).Methods(http.MethodPost)

	// resources
	r.PathPrefix(uri.DeviceResources).MatcherFunc(resourceMatcher).Methods(http.MethodPut).HandlerFunc(requestHandler.updateResource)
	r.PathPrefix(uri.DeviceResources).MatcherFunc(resourceMatcher).Methods(http.MethodGet).HandlerFunc(requestHandler.getResource)
	r.PathPrefix(uri.DeviceResources).MatcherFunc(resourceMatcher).Methods(http.MethodDelete).HandlerFunc(requestHandler.deleteResource)

	// ws
	r.PathPrefix(uri.WsStartDeviceResourceObservation).MatcherFunc(wsResourceMatcher).Methods(http.MethodGet).HandlerFunc(requestHandler.startResourceObservation)
	r.HandleFunc(uri.WsStartDevicesObservation, requestHandler.startDevicesObservation).Methods(http.MethodGet)
	r.HandleFunc(uri.WsStartDeviceResourcesObservation, requestHandler.startDeviceResourcesObservation).Methods(http.MethodGet)

	// serve www directory
	if requestHandler.config.UI.Enabled {
		r.HandleFunc(uri.OAuthConfiguration, requestHandler.getOAuthConfiguration).Methods(http.MethodGet)
		r.PathPrefix("/").Handler(http.FileServer(http.Dir(requestHandler.config.UI.Directory))).Methods(http.MethodGet)
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

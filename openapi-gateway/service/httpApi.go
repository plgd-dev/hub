package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"

	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/log"
	kitNetHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/ocf-cloud/openapi-connector/events"
	"github.com/go-ocf/ocf-cloud/openapi-gateway/store"
	"github.com/go-ocf/ocf-cloud/openapi-gateway/uri"

	raCqrs "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/notification"
	projectionRA "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs/projection"
	router "github.com/gorilla/mux"

	pbAS "github.com/go-ocf/ocf-cloud/authorization/pb"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/ocf-cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-shadow"
)

const resourceLinkHrefKey = "resourceLinkHref"
const subscriptionIDKey = "subscriptionID"
const deviceIDKey = "deviceID"

const ContentQuery = "content"
const ContentQueryBaseValue = "base"
const ContentQueryAllValue = "all"
const ContentQueryDefault = ContentQueryBaseValue

type ListDevicesOfUserFunc func(ctx context.Context, correlationID, userID, accessToken string) (deviceIds []string, statusCode int, err error)

//RequestHandler for handling incoming request
type RequestHandler struct {
	resourceProjection          *projectionRA.Projection
	store                       store.Store
	updateNotificationContainer *raCqrs.UpdateNotificationContainer
	timeoutForRequests          time.Duration

	asClient pbAS.AuthorizationServiceClient
	raClient pbRA.ResourceAggregateClient
	rsClient pbRS.ResourceShadowClient
	rdClient pbRD.ResourceDirectoryClient
	ddClient pbDD.DeviceDirectoryClient
}

func logAndWriteErrorResponse(err error, statusCode int, w http.ResponseWriter) {
	log.Errorf("%v", err)
	w.Header().Set(events.ContentTypeKey, "text/plain")
	w.WriteHeader(kitNetHttp.ErrToStatusWithDef(err, statusCode))
	w.Write([]byte(err.Error()))
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(
	asClient pbAS.AuthorizationServiceClient,
	raClient pbRA.ResourceAggregateClient,
	rsClient pbRS.ResourceShadowClient,
	rdClient pbRD.ResourceDirectoryClient,
	ddClient pbDD.DeviceDirectoryClient,
	resourceProjection *projectionRA.Projection,
	store store.Store,
	updateNotificationContainer *raCqrs.UpdateNotificationContainer,
	timeoutForRequests time.Duration,
) *RequestHandler {
	return &RequestHandler{
		asClient:                    asClient,
		raClient:                    raClient,
		rsClient:                    rsClient,
		rdClient:                    rdClient,
		ddClient:                    ddClient,
		resourceProjection:          resourceProjection,
		store:                       store,
		updateNotificationContainer: updateNotificationContainer,
		timeoutForRequests:          timeoutForRequests,
	}
}

func getContentQueryValue(u *url.URL) string {
	m, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ContentQueryDefault
	}

	c, _ := m[ContentQuery]
	if len(c) != 1 {
		return ContentQueryDefault
	}
	switch c[0] {
	case ContentQueryAllValue:
		return ContentQueryAllValue
	case ContentQueryBaseValue:
		return ContentQueryBaseValue
	}

	return ContentQueryDefault
}

const applicationMimeType = "application"

func getResponseWriterEncoder(accept []string) (responseWriterEncoderFunc, error) {
	if len(accept) == 0 {
		return newCBORResponseWriterEncoder(coap.AppOcfCbor.String()), nil
	}
	var encode responseWriterEncoderFunc
	for _, v := range accept {
		switch v {
		case coap.AppJSON.String():
			encode = jsonResponseWriterEncoder
		case coap.AppCBOR.String():
			return newCBORResponseWriterEncoder(v), nil
		case coap.AppOcfCbor.String():
			return newCBORResponseWriterEncoder(v), nil
		case applicationMimeType + "/*":
			return newCBORResponseWriterEncoder(coap.AppOcfCbor.String()), nil
		}
	}
	if encode != nil {
		return encode, nil
	}
	return nil, fmt.Errorf("invalid accept header(%v)", accept)
}

func getEncoder(contentType string) (func(v interface{}) ([]byte, error), error) {
	switch contentType {
	case events.ContentType_JSON:
		return json.Encode, nil
	case events.ContentType_VNDOCFCBOR:
		return cbor.Encode, nil
	}

	return nil, fmt.Errorf("invalid %v header(%v)", events.ContentTypeKey, contentType)
}

func makeResourceLinkHref(path []string) string {
	return "/" + strings.Join(path, "/")
}

func splitDevicePath(requestURI string) []string {
	p := kitNetHttp.CanonicalHref(requestURI)
	p = strings.TrimPrefix(p, uri.Devices) // remove core prefix
	p = strings.TrimLeft(p, "/")
	return strings.Split(p, "/")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("%v %v", r.Method, r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func resourceSubscriptionMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI)
	if len(paths) > 3 && paths[len(paths)-2] == "subscriptions" {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[deviceIDKey] = paths[0]
		rm.Vars[resourceLinkHrefKey] = makeResourceLinkHref(paths[1 : len(paths)-2])
		rm.Vars[subscriptionIDKey] = paths[len(paths)-1]
		return true
	}
	return false
}

func resourceMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI)
	if len(paths) >= 2 && paths[len(paths)-1] != "subscriptions" {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[deviceIDKey] = paths[0]
		rm.Vars[resourceLinkHrefKey] = makeResourceLinkHref(paths[1:])
		return true
	}
	return false
}

// NewHTTP returns HTTP server
func NewHTTP(requestHandler *RequestHandler, authInterceptor kitNetHttp.Interceptor) *http.Server {
	r := router.NewRouter()
	r.StrictSlash(true)
	r.Use(loggingMiddleware)
	r.Use(kitNetHttp.CreateAuthMiddleware(authInterceptor, func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		logAndWriteErrorResponse(fmt.Errorf("cannot process request on %v: %w", r.RequestURI, err), http.StatusUnauthorized, w)
	}))

	// health check
	r.HandleFunc("/", healthCheck).Methods("GET")

	s := r.PathPrefix(uri.Devices).Subrouter()

	// retrieve all devices
	s.HandleFunc("", requestHandler.RetrieveDevices).Methods("GET")

	// devices subscription
	s.HandleFunc("/subscriptions", requestHandler.SubscribeToDevices).Methods("POST")
	s.HandleFunc("/subscriptions/{"+subscriptionIDKey+"}", requestHandler.RetrieveDevicesSubscription).Methods("GET")
	s.HandleFunc("/subscriptions/{"+subscriptionIDKey+"}", requestHandler.UnsubscribeFromDevices).Methods("DELETE")

	// retrieve device
	s1 := s.PathPrefix("/").Subrouter()
	s1.HandleFunc("/{"+deviceIDKey+"}", requestHandler.RetrieveDevice).Methods("GET")

	// device subscription
	s1.HandleFunc("/{"+deviceIDKey+"}/subscriptions", requestHandler.SubscribeToDevice).Methods("POST")
	s1.HandleFunc("/{"+deviceIDKey+"}/subscriptions/{"+subscriptionIDKey+"}", requestHandler.RetrieveDeviceSubscription).Methods("GET")
	s1.HandleFunc("/{"+deviceIDKey+"}/subscriptions/{"+subscriptionIDKey+"}", requestHandler.UnsubscribeFromDevice).Methods("DELETE")

	// resource subscription
	s1.MatcherFunc(func(r *http.Request, rm *router.RouteMatch) bool {
		paths := splitDevicePath(r.RequestURI)
		if len(paths) > 2 && paths[len(paths)-1] == "subscriptions" {
			if rm.Vars == nil {
				rm.Vars = make(map[string]string)
			}
			rm.Vars[deviceIDKey] = paths[0]
			rm.Vars[resourceLinkHrefKey] = makeResourceLinkHref(paths[1 : len(paths)-1])
			return true
		}
		return false
	}).Methods("POST").HandlerFunc(requestHandler.SubscribeToResource)
	s1.MatcherFunc(resourceSubscriptionMatcher).Methods("GET").HandlerFunc(requestHandler.RetrieveResourceSubscription)
	s1.MatcherFunc(resourceSubscriptionMatcher).Methods("DELETE").HandlerFunc(requestHandler.UnsubscribeFromResource)

	// resource
	s1.MatcherFunc(resourceMatcher).Methods("POST").HandlerFunc(requestHandler.UpdateResource)
	s1.MatcherFunc(resourceMatcher).Methods("GET").HandlerFunc(requestHandler.RetrieveResource)
	return &http.Server{Handler: r}
}

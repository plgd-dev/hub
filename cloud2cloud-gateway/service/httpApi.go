package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	raClient "github.com/plgd-dev/hub/v2/resource-aggregate/client"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

const (
	hrefKey           = "Href"
	subscriptionIDKey = "subscriptionID"
	deviceIDKey       = "deviceID"
)

const (
	ContentQuery          = "content"
	ContentQueryBaseValue = "base"
	ContentQueryAllValue  = "all"
	ContentQueryDefault   = ContentQueryBaseValue
)

type ListDevicesOfUserFunc func(ctx context.Context, correlationID, userID, accessToken string) (deviceIds []string, statusCode int, err error)

// RequestHandler for handling incoming request
type RequestHandler struct {
	gwClient  pbGRPC.GrpcGatewayClient
	raClient  *raClient.Client
	subMgr    *SubscriptionManager
	emitEvent emitEventFunc
}

func logAndWriteErrorResponse(err error, statusCode int, w http.ResponseWriter) {
	log.Errorf("%v", err)
	w.Header().Set(events.ContentTypeKey, "text/plain")
	w.WriteHeader(pkgHttp.ErrToStatusWithDef(err, statusCode))
	if _, err2 := w.Write([]byte(err.Error())); err2 != nil {
		log.Errorf("failed to write error response body: %w", err2)
	}
}

// NewRequestHandler factory for new RequestHandler
func NewRequestHandler(
	gwClient pbGRPC.GrpcGatewayClient,
	raClient *raClient.Client,
	subMgr *SubscriptionManager,
	emitEvent emitEventFunc,
) *RequestHandler {
	return &RequestHandler{
		gwClient:  gwClient,
		raClient:  raClient,
		subMgr:    subMgr,
		emitEvent: emitEvent,
	}
}

func getContentQueryValue(u *url.URL) string {
	m, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ContentQueryDefault
	}

	c := m[ContentQuery]
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
		return newCBORResponseWriterEncoder(message.AppOcfCbor.String()), nil
	}
	var encode responseWriterEncoderFunc
	for _, v := range accept {
		switch v {
		case message.AppJSON.String():
			encode = jsonResponseWriterEncoder
		case message.AppCBOR.String():
			return newCBORResponseWriterEncoder(v), nil
		case message.AppOcfCbor.String():
			return newCBORResponseWriterEncoder(v), nil
		case applicationMimeType + "/*":
			return newCBORResponseWriterEncoder(message.AppOcfCbor.String()), nil
		}
	}
	if encode != nil {
		return encode, nil
	}
	return nil, fmt.Errorf("invalid accept header(%v)", accept)
}

func getEncoder(accept []string) (func(v interface{}) ([]byte, error), string, error) {
	for _, v := range accept {
		switch v {
		case events.ContentType_JSON:
			return json.Encode, events.ContentType_JSON, nil
		case events.ContentType_VNDOCFCBOR:
			return cbor.Encode, events.ContentType_VNDOCFCBOR, nil
		}
	}
	return nil, "", fmt.Errorf("invalid %v header(%v)", events.AcceptKey, accept)
}

func makeHref(path []string) string {
	return "/" + strings.Join(path, "/")
}

func splitDevicePath(requestURI string) []string {
	p := pkgHttpUri.CanonicalHref(requestURI)
	p = strings.TrimPrefix(p, uri.Devices) // remove core prefix
	p = strings.TrimLeft(p, "/")
	return strings.Split(p, "/")
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

const subscriptions = "subscriptions"

func resourceSubscriptionMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI)
	if len(paths) > 3 && paths[len(paths)-2] == subscriptions {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[deviceIDKey] = paths[0]
		rm.Vars[hrefKey] = makeHref(paths[1 : len(paths)-2])
		rm.Vars[subscriptionIDKey] = paths[len(paths)-1]
		return true
	}
	return false
}

func resourceMatcher(r *http.Request, rm *router.RouteMatch) bool {
	paths := splitDevicePath(r.RequestURI)
	if len(paths) >= 2 && paths[len(paths)-1] != subscriptions {
		if rm.Vars == nil {
			rm.Vars = make(map[string]string)
		}
		rm.Vars[deviceIDKey] = paths[0]
		rm.Vars[hrefKey] = makeHref(paths[1:])
		return true
	}
	return false
}

// NewHTTP returns HTTP handler
func NewHTTP(requestHandler *RequestHandler, authInterceptor pkgHttpJwt.Interceptor, logger log.Logger) http.Handler {
	r := router.NewRouter()
	r.StrictSlash(true)
	r.Use(pkgHttp.CreateLoggingMiddleware(pkgHttp.WithLogger(logger)))
	r.Use(pkgHttp.CreateAuthMiddleware(authInterceptor, func(_ context.Context, w http.ResponseWriter, r *http.Request, err error) {
		logAndWriteErrorResponse(fmt.Errorf("cannot process request on %v: %w", r.RequestURI, err), http.StatusUnauthorized, w)
	}))

	// health check
	r.HandleFunc("/", healthCheck).Methods(http.MethodGet)

	// retrieve all devices
	r.HandleFunc(uri.Devices, requestHandler.RetrieveDevices).Methods(http.MethodGet)

	// devices subscription
	r.HandleFunc(uri.DevicesSubscriptions, requestHandler.SubscribeToDevices).Methods(http.MethodPost)
	r.HandleFunc(uri.DevicesSubscription, requestHandler.RetrieveDevicesSubscription).Methods(http.MethodGet)
	r.HandleFunc(uri.DevicesSubscription, requestHandler.UnsubscribeFromDevices).Methods(http.MethodDelete)

	// retrieve device
	r.HandleFunc(uri.Device, requestHandler.RetrieveDevice).Methods(http.MethodGet)

	// device subscription
	r.HandleFunc(uri.DeviceSubscriptions, requestHandler.SubscribeToDevice).Methods(http.MethodPost)
	r.HandleFunc(uri.DeviceSubscription, requestHandler.RetrieveDeviceSubscription).Methods(http.MethodGet)
	r.HandleFunc(uri.DeviceSubscription, requestHandler.UnsubscribeFromDevice).Methods(http.MethodDelete)

	s1 := r.PathPrefix(uri.Device).Subrouter()
	// resource subscription
	s1.MatcherFunc(func(r *http.Request, rm *router.RouteMatch) bool {
		paths := splitDevicePath(r.RequestURI)
		if len(paths) > 2 && paths[len(paths)-1] == subscriptions {
			if rm.Vars == nil {
				rm.Vars = make(map[string]string)
			}
			rm.Vars[deviceIDKey] = paths[0]
			rm.Vars[hrefKey] = makeHref(paths[1 : len(paths)-1])
			return true
		}
		return false
	}).Methods(http.MethodPost).HandlerFunc(requestHandler.SubscribeToResource)
	s1.MatcherFunc(resourceSubscriptionMatcher).Methods(http.MethodGet).HandlerFunc(requestHandler.RetrieveResourceSubscription)
	s1.MatcherFunc(resourceSubscriptionMatcher).Methods(http.MethodDelete).HandlerFunc(requestHandler.UnsubscribeFromResource)

	// resource
	s1.MatcherFunc(resourceMatcher).Methods(http.MethodPost).HandlerFunc(requestHandler.UpdateResource)
	s1.MatcherFunc(resourceMatcher).Methods(http.MethodGet).HandlerFunc(requestHandler.RetrieveResource)
	return r
}

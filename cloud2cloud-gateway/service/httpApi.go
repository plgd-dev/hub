package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-gateway/uri"
	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	"github.com/plgd-dev/kit/log"

	router "github.com/gorilla/mux"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
)

const HrefKey = "Href"
const subscriptionIDKey = "subscriptionID"
const deviceIDKey = "deviceID"

const ContentQuery = "content"
const ContentQueryBaseValue = "base"
const ContentQueryAllValue = "all"
const ContentQueryDefault = ContentQueryBaseValue

type ListDevicesOfUserFunc func(ctx context.Context, correlationID, userID, accessToken string) (deviceIds []string, statusCode int, err error)

//RequestHandler for handling incoming request
type RequestHandler struct {
	rdClient  pbGRPC.GrpcGatewayClient
	raClient  *raClient.Client
	subMgr    *SubscriptionManager
	emitEvent emitEventFunc
}

func logAndWriteErrorResponse(err error, statusCode int, w http.ResponseWriter) {
	log.Errorf("%v", err)
	w.Header().Set(events.ContentTypeKey, "text/plain")
	w.WriteHeader(kitNetHttp.ErrToStatusWithDef(err, statusCode))
	if _, err2 := w.Write([]byte(err.Error())); err2 != nil {
		log.Errorf("failed to write error response body: %w", err2)
	}
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(
	rdClient pbGRPC.GrpcGatewayClient,
	raClient *raClient.Client,
	subMgr *SubscriptionManager,
	emitEvent emitEventFunc,
) *RequestHandler {
	return &RequestHandler{
		rdClient:  rdClient,
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
		rm.Vars[HrefKey] = makeHref(paths[1 : len(paths)-2])
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
		rm.Vars[HrefKey] = makeHref(paths[1:])
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
	}, true))

	// health check
	r.HandleFunc("/", healthCheck).Methods("GET")

	// retrieve all devices
	r.HandleFunc(uri.Devices, requestHandler.RetrieveDevices).Methods("GET")

	// devices subscription
	r.HandleFunc(uri.DevicesSubscriptions, requestHandler.SubscribeToDevices).Methods("POST")
	r.HandleFunc(uri.DevicesSubscription, requestHandler.RetrieveDevicesSubscription).Methods("GET")
	r.HandleFunc(uri.DevicesSubscription, requestHandler.UnsubscribeFromDevices).Methods("DELETE")

	// retrieve device
	r.HandleFunc(uri.Device, requestHandler.RetrieveDevice).Methods("GET")

	// device subscription
	r.HandleFunc(uri.DeviceSubscriptions, requestHandler.SubscribeToDevice).Methods("POST")
	r.HandleFunc(uri.DeviceSubscription, requestHandler.RetrieveDeviceSubscription).Methods("GET")
	r.HandleFunc(uri.DeviceSubscription, requestHandler.UnsubscribeFromDevice).Methods("DELETE")

	s1 := r.PathPrefix(uri.Device).Subrouter()
	// resource subscription
	s1.MatcherFunc(func(r *http.Request, rm *router.RouteMatch) bool {
		paths := splitDevicePath(r.RequestURI)
		if len(paths) > 2 && paths[len(paths)-1] == "subscriptions" {
			if rm.Vars == nil {
				rm.Vars = make(map[string]string)
			}
			rm.Vars[deviceIDKey] = paths[0]
			rm.Vars[HrefKey] = makeHref(paths[1 : len(paths)-1])
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

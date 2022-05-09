package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/go-coap/v2/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgOAuth2 "github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"go.opentelemetry.io/otel/trace"
)

const cloudIDKey = "CloudId"
const accountIDKey = "AccountId"

type provisionCacheData struct {
	linkedAccount store.LinkedAccount
	linkedCloud   store.LinkedCloud
}

// RequestHandler handles incoming requests
type RequestHandler struct {
	ownerClaim     string
	provider       *pkgOAuth2.PlgdProvider
	store          *Store
	provisionCache *cache.Cache
	subManager     *SubscriptionManager
	triggerTask    OnTaskTrigger
	tracerProvider trace.TracerProvider
}

func logAndWriteErrorResponse(err error, statusCode int, w http.ResponseWriter) {
	log.Errorf("%w", err)
	w.Header().Set(events.ContentTypeKey, "text/plain")
	w.WriteHeader(statusCode)
	if _, err2 := w.Write([]byte(err.Error())); err2 != nil {
		log.Errorf("failed to write error response body: %w", err2)
	}
}

func NewRequestHandler(
	ownerClaim string,
	provider *pkgOAuth2.PlgdProvider,
	subManager *SubscriptionManager,
	store *Store,
	triggerTask OnTaskTrigger,
	tracerProvider trace.TracerProvider,
) *RequestHandler {
	cache := cache.NewCache()
	add := periodic.New(subManager.devicesSubscription.ctx.Done(), time.Minute*5)
	add(func(now time.Time) bool {
		cache.CheckExpirations(now)
		return true
	})
	return &RequestHandler{
		ownerClaim:     ownerClaim,
		provider:       provider,
		subManager:     subManager,
		store:          store,
		provisionCache: cache,
		triggerTask:    triggerTask,
		tracerProvider: tracerProvider,
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// NewHTTP returns HTTP handler
func NewHTTP(requestHandler *RequestHandler, authInterceptor kitNetHttp.Interceptor) (http.Handler, error) {
	r := router.NewRouter()
	r.StrictSlash(true)
	r.Use(kitNetHttp.CreateLoggingMiddleware())
	r.Use(kitNetHttp.CreateAuthMiddleware(authInterceptor, func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		logAndWriteErrorResponse(fmt.Errorf("cannot process request on %v: %w", r.RequestURI, err), http.StatusUnauthorized, w)
	}))

	// health check
	r.HandleFunc("/", healthCheck).Methods(http.MethodGet)

	// retrieve all linked clouds
	r.HandleFunc(uri.LinkedClouds, requestHandler.RetrieveLinkedClouds).Methods(http.MethodGet)
	// add linked cloud
	r.HandleFunc(uri.LinkedClouds, requestHandler.AddLinkedCloud).Methods(http.MethodPost)
	// delete linked cloud
	r.HandleFunc(uri.LinkedCloud, requestHandler.DeleteLinkedCloud).Methods(http.MethodDelete)
	// add linked account
	r.HandleFunc(uri.LinkedAccounts, requestHandler.AddLinkedAccount).Methods(http.MethodGet)
	// delete linked cloud
	r.HandleFunc(uri.LinkedAccount, requestHandler.DeleteLinkedAccount).Methods(http.MethodDelete)
	// notify linked cloud
	r.HandleFunc(uri.Events, requestHandler.ProcessEvent).Methods(http.MethodPost)

	r.HandleFunc(uri.OAuthCallback, requestHandler.OAuthCallback).Methods(http.MethodGet, http.MethodPost)

	return r, nil
}

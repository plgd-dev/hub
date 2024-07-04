package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	router "github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config                  *Config
	authRestriction         *cache.Cache[string, struct{}]
	accessTokenKey          interface{}
	accessTokenJwkKey       jwk.Key
	refreshRestriction      *cache.Cache[string, struct{}]
	privateKeyJWTValidators map[string]*validator.Validator
}

// NewRequestHandler factory for new RequestHandler
func NewRequestHandler(ctx context.Context, config *Config, accessTokenKey interface{}, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*RequestHandler, func(), error) {
	accessTokenJwkKey, err := pkgJwt.CreateJwkKey(accessTokenKey)
	var closer fn.FuncList
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create jwk for idToken: %w", err)
	}
	authRestriction := cache.NewCache[string, struct{}]()
	refreshRestriction := cache.NewCache[string, struct{}]()
	add := periodic.New(ctx.Done(), time.Second*5)
	add(func(now time.Time) bool {
		authRestriction.CheckExpirations(now)
		refreshRestriction.CheckExpirations(now)
		return true
	})

	privateKeyJWTValidators := make(map[string]*validator.Validator, len(config.OAuthSigner.Clients))
	for _, c := range config.OAuthSigner.Clients {
		if !c.JWTPrivateKey.Enabled {
			continue
		}
		validator, err := validator.New(ctx, c.JWTPrivateKey.Authorization, fileWatcher, logger, tracerProvider)
		if err != nil {
			closer.Execute()
			return nil, nil, fmt.Errorf("cannot create validator: %w", err)
		}
		privateKeyJWTValidators[c.ID] = validator
		closer.AddFunc(validator.Close)
	}

	return &RequestHandler{
		config:                  config,
		authRestriction:         authRestriction,
		accessTokenJwkKey:       accessTokenJwkKey,
		accessTokenKey:          accessTokenKey,
		refreshRestriction:      refreshRestriction,
		privateKeyJWTValidators: privateKeyJWTValidators,
	}, closer.Execute, nil
}

// NewHTTP returns HTTP handler
func NewHTTP(requestHandler *RequestHandler, logger log.Logger) http.Handler {
	r := router.NewRouter()
	r.Use(kitHttp.CreateLoggingMiddleware(kitHttp.WithLogger(logger)))
	r.StrictSlash(true)

	r.HandleFunc(uri.JWKs, requestHandler.getJWKs).Methods(http.MethodGet)
	r.HandleFunc(uri.OpenIDConfiguration, requestHandler.getOpenIDConfiguration).Methods(http.MethodGet)
	r.HandleFunc(uri.Token, requestHandler.postToken).Methods(http.MethodPost)

	return r
}

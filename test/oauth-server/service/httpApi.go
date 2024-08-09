package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	router "github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
)

// RequestHandler for handling incoming request
type RequestHandler struct {
	config             *Config
	authSession        *cache.Cache[string, authorizedSession]
	authRestriction    *cache.Cache[string, struct{}]
	idTokenKey         *rsa.PrivateKey
	idTokenJwkKey      jwk.Key
	accessTokenKey     interface{}
	accessTokenJwkKey  jwk.Key
	refreshRestriction *cache.Cache[string, struct{}]
}

// NewRequestHandler factory for new RequestHandler
func NewRequestHandler(ctx context.Context, config *Config, idTokenKey *rsa.PrivateKey, accessTokenKey interface{}) (*RequestHandler, error) {
	idTokenJwkKey, err := pkgJwt.CreateJwkKey(idTokenKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create jwk for idToken: %w", err)
	}
	accessTokenJwkKey, err := pkgJwt.CreateJwkKey(accessTokenKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create jwk for idToken: %w", err)
	}
	authSession := cache.NewCache[string, authorizedSession]()
	authRestriction := cache.NewCache[string, struct{}]()
	refreshRestriction := cache.NewCache[string, struct{}]()
	add := periodic.New(ctx.Done(), time.Second*5)
	add(func(now time.Time) bool {
		authSession.CheckExpirations(now)
		authRestriction.CheckExpirations(now)
		refreshRestriction.CheckExpirations(now)
		return true
	})

	return &RequestHandler{
		config:             config,
		authSession:        authSession,
		authRestriction:    authRestriction,
		idTokenKey:         idTokenKey,
		idTokenJwkKey:      idTokenJwkKey,
		accessTokenJwkKey:  accessTokenJwkKey,
		accessTokenKey:     accessTokenKey,
		refreshRestriction: refreshRestriction,
	}, nil
}

// NewHTTP returns HTTP handler
func NewHTTP(requestHandler *RequestHandler, logger log.Logger) http.Handler {
	r := router.NewRouter()
	r.Use(pkgHttp.CreateLoggingMiddleware(pkgHttp.WithLogger(logger)))
	r.StrictSlash(true)

	// get JWKs
	r.HandleFunc(uri.JWKs, requestHandler.getJWKs).Methods(http.MethodGet)
	r.HandleFunc(uri.OpenIDConfiguration, requestHandler.getOpenIDConfiguration).Methods(http.MethodGet)

	r.HandleFunc(uri.Authorize, requestHandler.authorize)
	r.HandleFunc(uri.Token, requestHandler.tokenOptions).Methods(http.MethodOptions)
	r.HandleFunc(uri.Token, requestHandler.postToken).Methods(http.MethodPost)
	r.HandleFunc(uri.Token, requestHandler.getToken).Methods(http.MethodGet)
	r.HandleFunc(uri.UserInfo, requestHandler.getUserInfo).Methods(http.MethodGet)
	r.HandleFunc(uri.LogOut, requestHandler.logOut)

	return r
}

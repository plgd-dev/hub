package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	router "github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
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

func createJwkKey(privateKey interface{}) (jwk.Key, error) {
	var alg string
	var publicKey interface{}
	switch v := privateKey.(type) {
	case *rsa.PrivateKey:
		alg = jwa.RS256.String()
		publicKey = &v.PublicKey
	case *ecdsa.PrivateKey:
		alg = jwa.ES256.String()
		publicKey = &v.PublicKey
	}

	jwkKey, err := jwk.New(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwk: %w", err)
	}
	data, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal public key: %w", err)
	}

	if err = jwkKey.Set(jwk.KeyIDKey, uuid.NewSHA1(uuid.NameSpaceX500, data).String()); err != nil {
		return nil, setKeyError(jwk.KeyIDKey, err)
	}
	if err = jwkKey.Set(jwk.AlgorithmKey, alg); err != nil {
		return nil, setKeyError(jwk.AlgorithmKey, err)
	}
	return jwkKey, nil
}

// NewRequestHandler factory for new RequestHandler
func NewRequestHandler(ctx context.Context, config *Config, idTokenKey *rsa.PrivateKey, accessTokenKey interface{}) (*RequestHandler, error) {
	idTokenJwkKey, err := createJwkKey(idTokenKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create jwk for idToken: %w", err)
	}
	accessTokenJwkKey, err := createJwkKey(accessTokenKey)
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
	r.Use(kitHttp.CreateLoggingMiddleware(kitHttp.WithLogger(logger)))
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

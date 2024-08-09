package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/go-coap/v3/pkg/runner/periodic"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
)

type KeyCacheI interface {
	GetOrFetchKey(token *jwt.Token) (interface{}, error)
	GetOrFetchKeyWithContext(ctx context.Context, token *jwt.Token) (interface{}, error)
}

type Validator struct {
	keys        KeyCacheI
	verifyTrust bool
	tokenCache  *TokenCache
}

var (
	ErrMissingToken      = errors.New("missing token")
	ErrCannotParseToken  = errors.New("could not parse token")
	ErrCannotVerifyTrust = errors.New("could not verify token trust")
	ErrBlackListedToken  = errors.New("token is blacklisted")
)

type config struct {
	verifyTrust     bool
	clients         map[string]TokenIssuerClient
	cacheExpiration time.Duration
	stop            <-chan struct{}
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithTrustVerification(clients map[string]TokenIssuerClient, cacheExpiration time.Duration, stop <-chan struct{}) Option {
	return optionFunc(func(c *config) {
		c.verifyTrust = true
		c.clients = clients
		c.cacheExpiration = cacheExpiration
		c.stop = stop
	})
}

func NewValidator(keyCache KeyCacheI, logger log.Logger, opts ...Option) *Validator {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}
	v := &Validator{
		keys:        keyCache,
		verifyTrust: c.verifyTrust,
	}
	if c.verifyTrust && len(c.clients) > 0 {
		v.tokenCache = NewTokenCache(c.clients, c.cacheExpiration, logger)
		add := periodic.New(c.stop, c.cacheExpiration/2)
		add(func(now time.Time) bool {
			v.tokenCache.CheckExpirations(now)
			return true
		})
	}
	return v
}

func errParseToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotParseToken, err)
}

func errParseTokenInvalidClaimsType(t *jwt.Token) error {
	return fmt.Errorf("%w: unsupported type %T", ErrCannotParseToken, t.Claims)
}

func (v *Validator) checkTrust(ctx context.Context, token string, claims jwt.Claims) error {
	issuer, err := claims.GetIssuer()
	if err != nil {
		return err
	}
	issuer = pkgHttpUri.CanonicalURI(issuer)
	return v.tokenCache.VerifyTrust(ctx, issuer, token, claims)
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.ParseWithContext(context.Background(), token)
}

func errVerifyTrust(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotVerifyTrust, err)
}

func (v *Validator) ParseWithContext(ctx context.Context, token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, ErrMissingToken
	}

	jwtKeyfunc := func(token *jwt.Token) (interface{}, error) {
		return v.keys.GetOrFetchKeyWithContext(ctx, token)
	}

	t, err := jwt.Parse(token, jwtKeyfunc, jwt.WithIssuedAt())
	if err != nil {
		return nil, errParseToken(err)
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errParseTokenInvalidClaimsType(t)
	}
	if v.verifyTrust {
		if err = v.checkTrust(ctx, token, claims); err != nil {
			return nil, errVerifyTrust(err)
		}
	}
	return claims, nil
}

func (v *Validator) ParseWithClaims(ctx context.Context, token string, claims jwt.Claims) error {
	if token == "" {
		return ErrMissingToken
	}

	_, err := jwt.ParseWithClaims(token, claims, v.keys.GetOrFetchKey, jwt.WithIssuedAt())
	if err != nil {
		return errParseToken(err)
	}
	if v.verifyTrust {
		if err = v.checkTrust(ctx, token, claims); err != nil {
			return errVerifyTrust(err)
		}
	}
	return nil
}

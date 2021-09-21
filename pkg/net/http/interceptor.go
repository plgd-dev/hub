package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"

	"github.com/plgd-dev/cloud/pkg/security/jwt"
)

type Interceptor = func(ctx context.Context, method, uri string) (context.Context, error)

type AuthArgs struct {
	URI    *regexp.Regexp
	Scopes []*regexp.Regexp
}

// RequestMatcher allows request without token validation.
type RequestMatcher struct {
	Method string
	URI    *regexp.Regexp
}

// NewInterceptor authorizes HTTP request with validator.
func NewInterceptorWithValidator(validator Validator, auths map[string][]AuthArgs, whiteList ...RequestMatcher) Interceptor {
	validateJWT := validateJWTWithValidator(validator, MakeClaimsFunc(auths))
	return func(ctx context.Context, method, uri string) (context.Context, error) {
		for _, wa := range whiteList {
			if strings.EqualFold(method, wa.Method) && wa.URI.MatchString(uri) {
				return ctx, nil
			}
		}
		return validateJWT(ctx, method, uri)
	}
}

// NewInterceptor authorizes HTTP request.
func NewInterceptor(jwksURL string, tls *tls.Config, auths map[string][]AuthArgs, whiteList ...RequestMatcher) Interceptor {
	validateJWT := validateJWT(jwksURL, tls, MakeClaimsFunc(auths))
	return func(ctx context.Context, method, uri string) (context.Context, error) {
		for _, wa := range whiteList {
			if strings.EqualFold(method, wa.Method) && wa.URI.MatchString(uri) {
				return ctx, nil
			}
		}
		return validateJWT(ctx, method, uri)
	}
}

func MakeClaimsFunc(methods map[string][]AuthArgs) ClaimsFunc {
	return func(ctx context.Context, method, uri string) Claims {
		args, ok := methods[method]
		if !ok {
			return &DeniedClaims{fmt.Errorf("inaccessible method: %v", method)}
		}
		for _, arg := range args {
			if arg.URI.MatchString(uri) {
				return jwt.NewRegexpScopeClaims(arg.Scopes...)
			}
		}
		return &DeniedClaims{fmt.Errorf("inaccessible uri: %v %v", method, uri)}
	}
}

type DeniedClaims struct {
	Err error
}

func (c DeniedClaims) Valid() error {
	return c.Err
}

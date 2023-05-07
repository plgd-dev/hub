package http

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
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

func MakeClaimsFunc(methods map[string][]AuthArgs) ClaimsFunc {
	return func(ctx context.Context, method, uri string) jwt.ClaimsValidator {
		args, ok := methods[method]
		if !ok {
			return &DeniedClaims{Err: fmt.Errorf("inaccessible method: %v", method)}
		}
		for _, arg := range args {
			if arg.URI.MatchString(uri) {
				return pkgJwt.NewRegexpScopeClaims(arg.Scopes...)
			}
		}
		return &DeniedClaims{Err: fmt.Errorf("inaccessible uri: %v %v", method, uri)}
	}
}

type DeniedClaims struct {
	jwt.MapClaims
	Err error
}

func (c DeniedClaims) Validate() error {
	return c.Err
}

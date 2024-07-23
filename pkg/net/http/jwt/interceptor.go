package jwt

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type (
	ClaimsFunc = func(ctx context.Context, method, uri string) jwt.ClaimsValidator
	Validator  interface {
		ParseWithClaims(ctx context.Context, token string, claims jwt.Claims) error
	}
	Interceptor = func(ctx context.Context, method, uri string) (context.Context, error)
)

type AuthArgs struct {
	URI    *regexp.Regexp
	Scopes []*regexp.Regexp
}

// RequestMatcher allows request without token validation.
type RequestMatcher struct {
	Method string
	URI    *regexp.Regexp
}

func validateJWTWithValidator(validator Validator, claims ClaimsFunc) Interceptor {
	return func(ctx context.Context, method, uri string) (context.Context, error) {
		token, err := tokenFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		err = validator.ParseWithClaims(ctx, token, claims(ctx, method, uri))
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
		return ctx, nil
	}
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

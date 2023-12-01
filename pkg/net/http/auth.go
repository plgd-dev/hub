package http

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	ClaimsFunc               = func(ctx context.Context, method, uri string) jwt.ClaimsValidator
	OnUnauthorizedAccessFunc = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error)
	Validator                interface {
		ParseWithClaims(token string, claims jwt.Claims) error
	}
)

// NewDefaultAuthorizationRules returns a map of HTTP methods to a slice of AuthArgs.
// The AuthArgs contain a URI field that is a regular expression matching the given apiPath
// with any path suffix. This function is used to create default authorization rules for
// HTTP methods GET, POST, DELETE, and PUT.
func NewDefaultAuthorizationRules(apiPath string) map[string][]AuthArgs {
	return map[string][]AuthArgs{
		http.MethodGet: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodPost: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodDelete: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
		http.MethodPut: {
			{
				URI: regexp.MustCompile(regexp.QuoteMeta(apiPath) + AnyPathSuffixRegex),
			},
		},
	}
}

const (
	AnyPathSuffixRegex = `\/.*`

	bearerKey = "bearer"
)

type key int

const (
	authorizationKey key = 0
)

func ctxWithToken(ctx context.Context, token string) context.Context {
	if !strings.HasPrefix(strings.ToLower(token), bearerKey+" ") {
		token = fmt.Sprintf("%s %s", bearerKey, token)
	}
	return context.WithValue(ctx, authorizationKey, token)
}

func tokenFromCtx(ctx context.Context) (string, error) {
	val := ctx.Value(authorizationKey)
	if bearer, ok := val.(string); ok && strings.HasPrefix(strings.ToLower(bearer), bearerKey+" ") {
		token := bearer[7:]
		if token == "" {
			return "", status.Errorf(codes.Unauthenticated, "invalid token")
		}
		return token, nil
	}
	return "", status.Errorf(codes.Unauthenticated, "token not found")
}

func ParseToken(auth string) (string, error) {
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return auth[7:], nil
	}
	return "", status.Errorf(codes.Unauthenticated, "cannot parse bearer: prefix 'Bearer ' not found")
}

func validateJWTWithValidator(validator Validator, claims ClaimsFunc) Interceptor {
	return func(ctx context.Context, method, uri string) (context.Context, error) {
		token, err := tokenFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		err = validator.ParseWithClaims(token, claims(ctx, method, uri))
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
		return ctx, nil
	}
}

// CreateAuthMiddleware creates middleware for authorization
func CreateAuthMiddleware(authInterceptor Interceptor, onUnauthorizedAccessFunc OnUnauthorizedAccessFunc) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/": // health check
				next.ServeHTTP(w, r)
			default:
				token := r.Header.Get("Authorization")
				ctx := ctxWithToken(r.Context(), token)
				_, err := authInterceptor(ctx, r.Method, r.RequestURI)
				if err != nil {
					onUnauthorizedAccessFunc(ctx, w, r, err)
					return
				}

				if rawToken, err := ParseToken(token); err == nil {
					r = r.WithContext(grpc.CtxWithToken(r.Context(), rawToken))
				}
				next.ServeHTTP(w, r)
			}
		})
	}
}

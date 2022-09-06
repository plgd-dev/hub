package http

import (
	"context"
	"fmt"
	netHttp "net/http"
	"strings"

	extJwt "github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	Claims                   = interface{ Valid() error }
	ClaimsFunc               = func(ctx context.Context, method, uri string) Claims
	OnUnauthorizedAccessFunc = func(ctx context.Context, w netHttp.ResponseWriter, r *netHttp.Request, err error)
	Validator                interface {
		ParseWithClaims(token string, claims extJwt.Claims) error
	}
)

const bearerKey = "bearer"

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
func CreateAuthMiddleware(authInterceptor Interceptor, onUnauthorizedAccessFunc OnUnauthorizedAccessFunc) func(next netHttp.Handler) netHttp.Handler {
	return func(next netHttp.Handler) netHttp.Handler {
		return netHttp.HandlerFunc(func(w netHttp.ResponseWriter, r *netHttp.Request) {
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

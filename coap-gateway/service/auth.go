package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

type Claims = interface{ Valid() error }
type ClaimsFunc = func(ctx context.Context, code codes.Code, path string) Claims
type Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)

const bearerKey = "bearer"

type key int

const (
	authorizationKey key = 0
)

func CtxWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authorizationKey, fmt.Sprintf("%s %s", bearerKey, token))
}

func TokenFromCtx(ctx context.Context) (string, error) {
	val := ctx.Value(authorizationKey)
	if bearer, ok := val.(string); ok && strings.HasPrefix(bearer, bearerKey+" ") {
		token := strings.TrimPrefix(bearer, bearerKey+" ")
		if token == "" {
			return "", fmt.Errorf("invalid token")
		}
		return token, nil
	}
	return "", fmt.Errorf("token not found")
}

func ValidateJWT(jwksURL string, tls *tls.Config, claims ClaimsFunc) Interceptor {
	validator := jwt.NewValidator(jwksURL, tls)
	return func(ctx context.Context, code codes.Code, path string) (context.Context, error) {
		token, err := TokenFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		err = validator.ParseWithClaims(token, claims(ctx, code, path))
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
		return ctx, nil
	}
}

func NewAuthInterceptor() Interceptor {
	return func(ctx context.Context, code codes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SignUp, uri.SignIn:
			return ctx, nil
		}
		e := ctx.Value(&authCtxKey)
		if e == nil {
			return ctx, fmt.Errorf("invalid authorization context")
		}
		expire := e.(*authorizationContext)
		return ctx, expire.IsValid()
	}
}

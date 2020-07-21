package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/uri"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func isExpired(e time.Time) bool {
	return !e.IsZero() && time.Now().After(e)
}

func NewAuthInterceptor() kitNetCoap.Interceptor {
	return func(ctx context.Context, code coapCodes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SecureRefreshToken, uri.SignUp, uri.SecureSignUp, uri.SignIn, uri.SecureSignIn, uri.ResourcePing:
			return ctx, nil
		}
		_, err := kitNetCoap.TokenFromCtx(ctx)
		if err != nil {
			return ctx, err
		}
		e := ctx.Value(&expiredKey)
		if e == nil {
			return ctx, fmt.Errorf("invalid token expiration")
		}
		expire := e.(time.Time)
		if isExpired(expire) {
			return ctx, fmt.Errorf("token is expired")
		}
		return ctx, nil
	}
}

package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
)

func NewAuthInterceptor() kitNetCoap.Interceptor {
	return func(ctx context.Context, code coapCodes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SecureRefreshToken, uri.SignUp, uri.SecureSignUp, uri.SignIn, uri.SecureSignIn, uri.ResourcePing:
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

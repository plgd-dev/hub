package jwt

import (
	"context"
	"fmt"
	"strings"

	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type key int

const (
	authorizationKey key = 0

	bearerKey = "bearer"
)

func CtxWithToken(ctx context.Context, token string) context.Context {
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

func SubjectFromToken(token string) (string, bool) {
	claims, err := pkgJwt.ParseToken(token)
	if err != nil {
		return "", false
	}
	subject, err := claims.GetSubject()
	if err != nil {
		return "", false
	}
	return subject, true
}

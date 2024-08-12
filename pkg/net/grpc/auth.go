package grpc

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var authorizationKey = "authorization"

type AuthInterceptors struct {
	authFunc Interceptor
}

func MakeAuthInterceptors(authFunc Interceptor, whiteListedMethods ...string) AuthInterceptors {
	return AuthInterceptors{
		authFunc: func(ctx context.Context, method string) (context.Context, error) {
			for _, wa := range whiteListedMethods {
				if wa == method {
					return ctx, nil
				}
			}
			return authFunc(ctx, method)
		},
	}
}

func (f AuthInterceptors) Unary() grpc.UnaryServerInterceptor {
	return UnaryServerInterceptor(f.authFunc)
}

func (f AuthInterceptors) Stream() grpc.StreamServerInterceptor {
	return StreamServerInterceptor(f.authFunc)
}

type (
	ClaimsFunc = func(ctx context.Context, method string) jwt.ClaimsValidator
	Validator  interface {
		ParseWithClaims(ctx context.Context, token string, claims jwt.Claims) error
	}
)

func ValidateJWTWithValidator(validator Validator, claims ClaimsFunc) Interceptor {
	return func(ctx context.Context, method string) (context.Context, error) {
		token, err := grpc_auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		err = validator.ParseWithClaims(ctx, token, claims(ctx, method))
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}
		return ctx, nil
	}
}

// CtxWithToken stores token to ctx of request.
func CtxWithToken(ctx context.Context, token string) context.Context {
	niceMD := metautils.ExtractOutgoing(ctx)
	niceMD.Set(authorizationKey, fmt.Sprintf("%s %s", "bearer", token))
	return niceMD.ToOutgoing(ctx)
}

// CtxWithIncomingToken stores token to ctx of response.
func CtxWithIncomingToken(ctx context.Context, token string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(authorizationKey, fmt.Sprintf("%s %s", "bearer", token))
	return niceMD.ToIncoming(ctx)
}

func ParseOwnerFromJwtToken(ownerClaim, rawJwtToken string) (string, error) {
	claims, err := pkgJwt.ParseToken(rawJwtToken)
	if err != nil {
		return "", err
	}
	owner, err := claims.GetOwner(ownerClaim)
	if err != nil {
		return "", err
	}
	if owner == "" {
		return "", fmt.Errorf("claim '%v' was not found", ownerClaim)
	}

	return owner, nil
}

// OwnerFromTokenMD is a helper function for extracting the ownerClaim from the :authorization gRPC metadata of the request.
func OwnerFromTokenMD(ctx context.Context, ownerClaim string) (string, error) {
	accessToken, err := TokenFromMD(ctx)
	if err != nil {
		return "", err
	}
	owner, err := ParseOwnerFromJwtToken(ownerClaim, accessToken)
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	return owner, err
}

// SubjectFromTokenMD is a helper function for extracting the sub claim from the :authorization gRPC metadata of the request.
func SubjectFromTokenMD(ctx context.Context) (string, error) {
	token, err := TokenFromMD(ctx)
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	claims, err := pkgJwt.ParseToken(token)
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	subject, err := claims.GetSubject()
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	if subject == "" {
		return "", status.Errorf(codes.InvalidArgument, "invalid subject in token")
	}
	return subject, nil
}

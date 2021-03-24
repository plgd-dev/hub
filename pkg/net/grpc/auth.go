package grpc

import (
	"context"
	"crypto/tls"
	"fmt"

	extJwt "github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"

	"github.com/plgd-dev/cloud/pkg/security/jwt"
)

var authorizationKey = "authorization"
var ownerKey = "owner"

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

func MakeJWTInterceptors(jwksURL string, tls *tls.Config, claims ClaimsFunc, whiteListedMethods ...string) AuthInterceptors {
	return MakeAuthInterceptors(ValidateJWT(jwksURL, tls, claims), whiteListedMethods...)
}

func (f AuthInterceptors) Unary() grpc.UnaryServerInterceptor {
	return UnaryServerInterceptor(f.authFunc)
}
func (f AuthInterceptors) Stream() grpc.StreamServerInterceptor {
	return StreamServerInterceptor(f.authFunc)
}

type ClaimsFunc = func(ctx context.Context, method string) Claims
type Claims = interface{ Valid() error }

func ValidateJWT(jwksURL string, tls *tls.Config, claims ClaimsFunc) Interceptor {
	validator := jwt.NewValidator(jwksURL, tls)
	return func(ctx context.Context, method string) (context.Context, error) {
		token, err := grpc_auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		err = validator.ParseWithClaims(token, claims(ctx, method))
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

// CtxWithOwner stores owner to ctx of request.
func CtxWithOwner(ctx context.Context, owner string) context.Context {
	niceMD := metautils.ExtractOutgoing(ctx)
	niceMD.Set(ownerKey, owner)
	return niceMD.ToOutgoing(ctx)
}

// CtxWithIncomingToken stores token to ctx of response.
func CtxWithIncomingToken(ctx context.Context, token string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(authorizationKey, fmt.Sprintf("%s %s", "bearer", token))
	return niceMD.ToIncoming(ctx)
}

// CtxWithIncomingOwner stores owner to ctx of response.
func CtxWithIncomingOwner(ctx context.Context, owner string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(ownerKey, owner)
	return niceMD.ToIncoming(ctx)
}

// OwnerFromMD is a helper function for extracting the :userid header from the gRPC metadata of the request.
func OwnerFromMD(ctx context.Context) (string, error) {
	val := metautils.ExtractIncoming(ctx).Get(ownerKey)
	if val == "" {
		return "", status.Errorf(codes.InvalidArgument, "owner not found in request")
	}
	return val, nil
}

// OwnerFromOutgoingMD extracts owner stored by CtxWithOwner.
func OwnerFromOutgoingMD(ctx context.Context) (string, error) {
	val := metautils.ExtractOutgoing(ctx).Get(ownerKey)
	if val == "" {
		return "", status.Errorf(codes.InvalidArgument, "owner not found in request")
	}
	return val, nil
}

type claims map[string]interface{}

func (c *claims) Valid() error {
	return nil
}

func ParseOwnerFromJwtToken(ownerClaim, rawJwtToken string) (string, error) {
	parser := &extJwt.Parser{
		SkipClaimsValidation: true,
	}

	var claims claims
	_, _, err := parser.ParseUnverified(rawJwtToken, &claims)
	if err != nil {
		return "", err
	}

	ownerI, ok := claims[ownerClaim]
	if ok {
		if owner, ok := ownerI.(string); ok {
			return owner, nil
		}
	}

	return "", fmt.Errorf("claim '%v' was not found", ownerClaim)
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

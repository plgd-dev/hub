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
var userIDKey = "userid"

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
			return nil, grpc.Errorf(codes.Unauthenticated, "invalid token: %v", err)
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

// CtxWithUserID stores userID to ctx of request.
func CtxWithUserID(ctx context.Context, userID string) context.Context {
	niceMD := metautils.ExtractOutgoing(ctx)
	niceMD.Set(userIDKey, userID)
	return niceMD.ToOutgoing(ctx)
}

// CtxWithIncomingToken stores token to ctx of reponse.
func CtxWithIncomingToken(ctx context.Context, token string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(authorizationKey, fmt.Sprintf("%s %s", "bearer", token))
	return niceMD.ToIncoming(ctx)
}

// CtxWithIncomingUserID stores userID to ctx of reponse.
func CtxWithIncomingUserID(ctx context.Context, userID string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(userIDKey, userID)
	return niceMD.ToIncoming(ctx)
}

// UserIDFromMD is a helper function for extracting the :userid header from the gRPC metadata of the request.
func UserIDFromMD(ctx context.Context) (string, error) {
	val := metautils.ExtractIncoming(ctx).Get(userIDKey)
	if val == "" {
		return "", status.Errorf(codes.InvalidArgument, "UserID not found in request")
	}
	return val, nil
}

// UserIDFromOutgoingMD extracts userID stored by CtxWithUserID.
func UserIDFromOutgoingMD(ctx context.Context) (string, error) {
	val := metautils.ExtractOutgoing(ctx).Get(userIDKey)
	if val == "" {
		return "", status.Errorf(codes.InvalidArgument, "UserID not found in request")
	}
	return val, nil
}

type claims struct {
	Subject string `json:"sub,omitempty"`
}

func (c *claims) Valid() error {
	return nil
}

func parseSubFromJwtToken(rawJwtToken string) (string, error) {
	parser := &extJwt.Parser{
		SkipClaimsValidation: true,
	}

	var claims claims
	_, _, err := parser.ParseUnverified(rawJwtToken, &claims)
	if err != nil {
		return "", fmt.Errorf("cannot get subject from jwt token: %w", err)
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("cannot get subject from jwt token: not found")
}

// UserIDFromTokenMD is a helper function for extracting the userID from the :authorization gRPC metadata of the request.
func UserIDFromTokenMD(ctx context.Context) (string, error) {
	accessToken, err := TokenFromMD(ctx)
	if err != nil {
		return "", err
	}
	userID, err := parseSubFromJwtToken(accessToken)
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	return userID, err
}

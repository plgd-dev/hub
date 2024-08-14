package grpc

import (
	"context"
	"strings"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	headerAuthorize = "authorization"
)

func errUnauthenticated(scheme string) error {
	return status.Errorf(codes.Unauthenticated, "Request unauthenticated with %s", scheme)
}

// TokenFromOutgoingMD extracts token stored by CtxWithToken.
func TokenFromOutgoingMD(ctx context.Context) (string, error) {
	expectedScheme := "bearer"
	val := metautils.ExtractOutgoing(ctx).Get(headerAuthorize)
	if val == "" {
		return "", errUnauthenticated(expectedScheme)
	}
	splits := strings.SplitN(val, " ", 2)
	if len(splits) < 2 {
		return "", status.Errorf(codes.Unauthenticated, "Bad authorization string")
	}
	if !strings.EqualFold(splits[0], strings.ToLower(expectedScheme)) {
		return "", errUnauthenticated(expectedScheme)
	}
	return splits[1], nil
}

// TokenFromMD is a helper function for extracting the :authorization header from the gRPC metadata of the request.
func TokenFromMD(ctx context.Context) (string, error) {
	return grpc_auth.AuthFromMD(ctx, "bearer")
}

// OwnerFromOutgoingTokenMD extracts ownerClaim from token stored by CtxWithToken.
func OwnerFromOutgoingTokenMD(ctx context.Context, ownerClaim string) (string, error) {
	accessToken, err := TokenFromOutgoingMD(ctx)
	if err != nil {
		return "", err
	}
	owner, err := ParseOwnerFromJwtToken(ownerClaim, accessToken)
	if err != nil {
		return "", ForwardFromError(codes.InvalidArgument, err)
	}
	return owner, err
}

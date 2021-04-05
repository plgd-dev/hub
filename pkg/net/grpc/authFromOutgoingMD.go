package grpc

import (
	"context"
	"strings"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	headerAuthorize = "authorization"
)

// TokenFromOutgoingMD extracts token stored by CtxWithToken.
func TokenFromOutgoingMD(ctx context.Context) (string, error) {
	expectedScheme := "bearer"
	val := metautils.ExtractOutgoing(ctx).Get(headerAuthorize)
	if val == "" {
		return "", grpc.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)

	}
	splits := strings.SplitN(val, " ", 2)
	if len(splits) < 2 {
		return "", grpc.Errorf(codes.Unauthenticated, "Bad authorization string")
	}
	if strings.ToLower(splits[0]) != strings.ToLower(expectedScheme) {
		return "", grpc.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	return splits[1], nil
}

// TokenFromMD is a helper function for extracting the :authorization header from the gRPC metadata of the request.
func TokenFromMD(ctx context.Context) (string, error) {
	return grpc_auth.AuthFromMD(ctx, "bearer")
}

package grpc

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
)

var (
	AcceptContentHeaderKey = "accept-content"
)

// AcceptContentFromOutgoingMD extracts accept stored by CtxWithToken.
func AcceptContentFromOutgoingMD(ctx context.Context) string {
	return metautils.ExtractOutgoing(ctx).Get(AcceptContentHeaderKey)
}

// AcceptContentFromMD is a helper function for extracting the :accept header from the gRPC metadata of the request.
func AcceptContentFromMD(ctx context.Context) string {
	return metautils.ExtractIncoming(ctx).Get(AcceptContentHeaderKey)
}

// CtxWithOwner stores owner to ctx of request.
func CtxWithAcceptContent(ctx context.Context, accept string) context.Context {
	niceMD := metautils.ExtractOutgoing(ctx)
	niceMD.Set(AcceptContentHeaderKey, accept)
	return niceMD.ToOutgoing(ctx)
}

// CtxWithIncomingAcceptContent stores token to ctx of response.
func CtxWithIncomingAcceptContent(ctx context.Context, accept string) context.Context {
	niceMD := metautils.ExtractIncoming(ctx)
	niceMD.Set(AcceptContentHeaderKey, accept)
	return niceMD.ToIncoming(ctx)
}

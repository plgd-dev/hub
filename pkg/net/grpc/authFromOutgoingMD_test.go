package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthFromOutgoingMD(t *testing.T) {
	token := "token"
	got, err := TokenFromOutgoingMD(CtxWithToken(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestOwnerFromOutgoingMD(t *testing.T) {
	token := "token"
	got, err := OwnerFromOutgoingMD(CtxWithOwner(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestOwnerFromMD(t *testing.T) {
	token := "token"
	got, err := OwnerFromMD(CtxWithIncomingOwner(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestAuthIDFromMD(t *testing.T) {
	token := "token"
	got, err := TokenFromMD(CtxWithIncomingToken(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

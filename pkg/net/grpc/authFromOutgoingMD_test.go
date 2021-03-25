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

func TestUserIDFromOutgoingMD(t *testing.T) {
	token := "token"
	got, err := UserIDFromOutgoingMD(CtxWithUserID(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestUserIDFromMD(t *testing.T) {
	token := "token"
	got, err := UserIDFromMD(CtxWithIncomingUserID(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

func TestAuthIDFromMD(t *testing.T) {
	token := "token"
	got, err := TokenFromMD(CtxWithIncomingToken(context.Background(), token))
	require.NoError(t, err)
	require.Equal(t, token, got)
}

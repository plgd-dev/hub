package http

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCtxWithToken(t *testing.T) {
	ctx := context.Background()
	token, err := TokenFromCtx(CtxWithToken(ctx, "a"))
	require.NoError(t, err)
	require.Equal(t, "a", token)

	token, err = TokenFromCtx(CtxWithToken(ctx, bearerKey+" b"))
	require.NoError(t, err)
	require.Equal(t, "b", token)

	token, err = TokenFromCtx(CtxWithToken(ctx, "Bearer c"))
	require.NoError(t, err)
	require.Equal(t, "c", token)
}

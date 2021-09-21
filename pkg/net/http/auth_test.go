package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCtxWithToken(t *testing.T) {
	ctx := context.Background()
	token, err := tokenFromCtx(ctxWithToken(ctx, "a"))
	require.NoError(t, err)
	require.Equal(t, "a", token)

	token, err = tokenFromCtx(ctxWithToken(ctx, bearerKey+" b"))
	require.NoError(t, err)
	require.Equal(t, "b", token)

	token, err = tokenFromCtx(ctxWithToken(ctx, "Bearer c"))
	require.NoError(t, err)
	require.Equal(t, "c", token)
}

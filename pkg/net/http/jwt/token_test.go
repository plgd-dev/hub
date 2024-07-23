package jwt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCtxWithToken(t *testing.T) {
	ctx := context.Background()
	token, err := tokenFromCtx(CtxWithToken(ctx, "a"))
	require.NoError(t, err)
	require.Equal(t, "a", token)

	token, err = tokenFromCtx(CtxWithToken(ctx, bearerKey+" b"))
	require.NoError(t, err)
	require.Equal(t, "b", token)

	token, err = tokenFromCtx(CtxWithToken(ctx, "Bearer c"))
	require.NoError(t, err)
	require.Equal(t, "c", token)
}

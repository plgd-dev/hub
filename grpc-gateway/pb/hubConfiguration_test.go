package pb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthClient_Clone(t *testing.T) {
	a := OAuthClient{
		ClientId: "test",
		Scopes:   []string{"test"},
	}
	b := a.Clone()
	require.Equal(t, &a, b)
	require.NotEqual(t, fmt.Sprintf("%p", &a.Scopes), fmt.Sprintf("%p", b.GetScopes()))
}

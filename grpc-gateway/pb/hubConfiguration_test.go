package pb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceOAuthClient_Clone(t *testing.T) {
	a := DeviceOAuthClient{
		ProviderName: "test",
		Scopes:       []string{"test"},
	}
	b := a.Clone()
	require.Equal(t, &a, b)
	require.NotEqual(t, fmt.Sprintf("%p", &a.Scopes), fmt.Sprintf("%p", b.GetScopes()))
}

func TestWebOAuthClient_Clone(t *testing.T) {
	a := WebOAuthClient{
		ClientId: "test",
		Scopes:   []string{"test"},
	}
	b := a.Clone()
	require.Equal(t, &a, b)
	require.NotEqual(t, fmt.Sprintf("%p", &a.Scopes), fmt.Sprintf("%p", b.GetScopes()))
}

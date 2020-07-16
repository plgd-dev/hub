package refImpl

import (
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	var config Config
	os.Setenv("OAUTH_CALLBACK", "OAUTH_CALLBACK")
	os.Setenv("EVENTS_URL", "EVENTS_URL")
	os.Setenv("NAME", "NAME")
	os.Setenv("CLIENT_ID", "CLIENT_ID")
	os.Setenv("CLIENT_SECRET", "CLIENT_SECRET")
	os.Setenv("SCOPES", "SCOPES")
	os.Setenv("AUTH_URL", "AUTH_URL")
	os.Setenv("TOKEN_URL", "TOKEN_URL")
	err := envconfig.Process("", &config)
	require.NoError(t, err)

	got, err := Init(config)
	require.NoError(t, err)
	require.NotEmpty(t, got)
}
